package httpapi

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"kvmm/internal/config"
	"kvmm/internal/handler/community"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/handler/posts"
	"kvmm/internal/handler/relationship"
	"kvmm/internal/logging"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// обработчики приложения
type Dependencies struct {
	Config       config.Config
	Feed         *feed.FeedHandler
	Posts        *posts.PostHandler
	Communities  *community.Handler
	Relationship *relationship.FriendsHandler
}

// NewRouter создаёт маршрутизатор API на основе gorilla/mux
func NewRouter(deps Dependencies) http.Handler {
	appConfig := deps.Config
	if appConfig.HTTP.Addr == "" {
		appConfig = config.Default()
	}
	router := mux.NewRouter()

	if deps.Feed != nil {
		router.HandleFunc("/api/feed", deps.Feed.List).Methods(http.MethodGet)
	}
	if deps.Posts != nil {
		router.HandleFunc("/api/posts/create", deps.Posts.Create).Methods(http.MethodPost)
		router.HandleFunc("/api/posts/delete", deps.Posts.Delete).Methods(http.MethodDelete)
		router.HandleFunc("/api/posts/update", deps.Posts.Update).Methods(http.MethodPatch)
		router.HandleFunc("/api/posts/user", deps.Posts.UserPosts).Methods(http.MethodGet)
		router.HandleFunc("/api/posts/{uid}/like", deps.Posts.Like).Methods(http.MethodPut)
		router.HandleFunc("/api/posts/{uid}/like", deps.Posts.Unlike).Methods(http.MethodDelete)
		router.HandleFunc("/api/posts/{uid}/likes", deps.Posts.Likers).Methods(http.MethodGet)
		router.HandleFunc("/api/community-posts/create", deps.Posts.CreateCommunity).Methods(http.MethodPost)
		router.HandleFunc("/api/community-posts/update", deps.Posts.UpdateCommunity).Methods(http.MethodPatch)
		router.HandleFunc("/api/community-posts/delete", deps.Posts.DeleteCommunity).Methods(http.MethodDelete)
		router.HandleFunc("/api/community-posts/list", deps.Posts.ListCommunity).Methods(http.MethodGet)
	}
	if deps.Communities != nil {
		router.HandleFunc("/api/communities", deps.Communities.Create).Methods(http.MethodPost)
		router.HandleFunc("/api/communities", deps.Communities.List).Methods(http.MethodGet)
		router.HandleFunc("/api/communities/subscriptions", deps.Communities.Subscriptions).Methods(http.MethodGet)
		router.HandleFunc("/api/communities/{uid}/subscribe", deps.Communities.Subscribe).Methods(http.MethodPost)
		router.HandleFunc("/api/communities/{uid}/subscribe", deps.Communities.Unsubscribe).Methods(http.MethodDelete)
		router.HandleFunc("/api/communities/{uid}/admins", deps.Communities.AssignAdmin).Methods(http.MethodPost)
		router.HandleFunc("/api/communities/{uid}", deps.Communities.Read).Methods(http.MethodGet)
		router.HandleFunc("/api/communities/{uid}", deps.Communities.Update).Methods(http.MethodPatch)
		router.HandleFunc("/api/communities/{uid}", deps.Communities.Delete).Methods(http.MethodDelete)
	}
	if deps.Relationship != nil {
		router.HandleFunc("/api/relationships/list", deps.Relationship.List).Methods(http.MethodGet)
		router.HandleFunc("/api/relationships/typeslist", deps.Relationship.TypesList).Methods(http.MethodGet)
		router.HandleFunc("/api/relationships/create", deps.Relationship.Create).Methods(http.MethodPost)
		router.HandleFunc("/api/relationships/cancle", deps.Relationship.Cancle).Methods(http.MethodDelete)
		router.HandleFunc("/api/relationships/accept", deps.Relationship.Accept).Methods(http.MethodPatch)
		router.HandleFunc("/api/relationships/requests", deps.Relationship.Requests).Methods(http.MethodGet)
		router.HandleFunc("/api/relationships/suggestions", deps.Relationship.Suggestions).Methods(http.MethodGet)
	}

	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			logging.Logger().Error("health response write failed", zap.Error(err))
		}
	}).Methods(http.MethodGet)

	return requestLogger(cors(router, appConfig.Frontend.Origin))
}

type responseLogger struct {
	http.ResponseWriter
	requestID   string
	status      int
	wroteHeader bool
}

func (w *responseLogger) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseLogger) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseLogger) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	flusher.Flush()
}

func (w *responseLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	w.status = http.StatusSwitchingProtocols
	w.wroteHeader = true
	return hijacker.Hijack()
}

func (w *responseLogger) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// RequestID возвращает ID текущего HTTP-запроса для вспомогательных логов
func (w *responseLogger) RequestID() string {
	return w.requestID
}

// requestLogger записывает входящие HTTP-запросы
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		requestID := logging.NewRequestID()
		r = r.WithContext(logging.WithRequestID(r.Context(), requestID))
		response := &responseLogger{ResponseWriter: w, requestID: requestID, status: http.StatusOK}
		response.Header().Set(logging.RequestIDHeader, requestID)
		requestBody := readRequestBodyForLog(r)

		defer func() {
			if recovered := recover(); recovered != nil {
				logging.Logger().Error("panic recovered in HTTP handler",
					zap.Any("panic", recovered),
					zap.String("request_id", requestID),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
				)
				if !response.wroteHeader {
					helpers.WriteError(response, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("request_id", requestID),
				zap.Int("status", response.status),
				zap.Duration("duration", time.Since(startedAt)),
			}
			if response.status >= http.StatusInternalServerError {
				logging.Logger().Error("HTTP request completed", fields...)
			} else {
				logging.Logger().Info("HTTP request completed", fields...)
			}
			logging.Logger().Debug("HTTP request details",
				zap.String("request_id", requestID),
				zap.String("query", r.URL.RawQuery),
				zap.String("user_agent", r.UserAgent()),
			)
			if requestBody != "" {
				logging.Logger().Debug("HTTP request body",
					zap.String("request_id", requestID),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("body", requestBody),
				)
			}
		}()

		next.ServeHTTP(response, r)
	})
}

const maxLoggedRequestBodySize = 1 << 20

// readRequestBodyForLog читает тело запроса для debug-лога и возвращает его обработчику
func readRequestBodyForLog(r *http.Request) string {
	if !logging.Logger().Core().Enabled(zap.DebugLevel) || r.Body == nil || r.Body == http.NoBody {
		return ""
	}

	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		logging.Logger().Debug("HTTP request body read failed", zap.Error(err))
		return "[body read failed]"
	}
	if len(body) == 0 {
		return ""
	}
	if len(body) > maxLoggedRequestBodySize {
		return "[body omitted: exceeds 1 MiB]"
	}

	contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if contentType == "multipart/form-data" {
		return "[multipart body omitted]"
	}
	if contentType == "application/json" || strings.HasSuffix(contentType, "+json") ||
		strings.HasPrefix(contentType, "text/") || contentType == "application/x-www-form-urlencoded" {
		return string(body)
	}
	return "[binary body omitted]"
}

// cors добавляет CORS-заголовки и обрабатывает preflight-запросы
func cors(next http.Handler, configuredOrigin string) http.Handler {
	allowedOrigin := configuredOrigin
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8001"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (origin == allowedOrigin || allowedOrigin == "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", logging.RequestIDHeader)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
