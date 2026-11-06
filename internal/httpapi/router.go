package httpapi

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/chat"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/handler/posts"
	"kvmm/internal/handler/profile"
	"kvmm/internal/handler/relationship"
	"kvmm/internal/logging"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// обработчики приложения
type Dependencies struct {
	Auth         *auth.AuthHandler
	Chat         *chat.ChatHandler
	Feed         *feed.FeedHandler
	Posts        *posts.PostHandler
	Profile      *profile.ProfileHandler
	Relationship *relationship.FriendsHandler
}

// NewRouter создаёт маршрутизатор API на основе gorilla/mux
func NewRouter(deps Dependencies) http.Handler {
	router := mux.NewRouter()

	authHandler := deps.Auth
	chatHandler := deps.Chat
	feedHandler := deps.Feed
	postHandler := deps.Posts
	profileHandler := deps.Profile
	relationshipHandler := deps.Relationship

	router.HandleFunc("/api/auth/register", authHandler.Register).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/me", authHandler.Me).Methods(http.MethodGet)
	router.HandleFunc("/api/auth/logout", authHandler.Logout).Methods(http.MethodPost)

	router.HandleFunc("/api/feed", feedHandler.List).Methods(http.MethodGet)

	router.HandleFunc("/api/posts/create", postHandler.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/posts/delete", postHandler.Delete).Methods(http.MethodDelete)
	router.HandleFunc("/api/posts/update", postHandler.Update).Methods(http.MethodPatch)
	router.HandleFunc("/api/posts/user", postHandler.UserPosts).Methods(http.MethodGet)

	mediaDir := os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "./internal/store/media"
	}
	router.PathPrefix("/api/media/").Handler(
		http.StripPrefix("/api/media/", http.FileServer(http.Dir(mediaDir))),
	)

	router.HandleFunc("/api/profile/update", profileHandler.Update).Methods(http.MethodPatch)
	router.HandleFunc("/api/profile/avatar", profileHandler.UploadAvatar).Methods(http.MethodPost)
	router.HandleFunc("/api/profile/avatars", profileHandler.ListAvatars).Methods(http.MethodGet)
	router.HandleFunc("/api/profile/avatars/{id}", profileHandler.DeleteAvatar).Methods(http.MethodDelete)

	router.HandleFunc("/api/chats/list", chatHandler.List).Methods(http.MethodGet)
	router.HandleFunc("/api/chats/messages", chatHandler.Messages).Methods(http.MethodGet)
	router.HandleFunc("/api/chats/send", chatHandler.Send).Methods(http.MethodPost)
	router.HandleFunc("/api/chats/create", chatHandler.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/chats/ws", chatHandler.WebSocket).Methods(http.MethodGet)

	router.HandleFunc("/api/relationships/list", relationshipHandler.List).Methods(http.MethodGet)
	router.HandleFunc("/api/relationships/typeslist", relationshipHandler.TypesList).Methods(http.MethodGet)
	router.HandleFunc("/api/relationships/create", relationshipHandler.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/relationships/cancle", relationshipHandler.Cancle).Methods(http.MethodDelete)
	router.HandleFunc("/api/relationships/accept", relationshipHandler.Accept).Methods(http.MethodPatch)
	router.HandleFunc("/api/relationships/requests", relationshipHandler.Requests).Methods(http.MethodGet)
	router.HandleFunc("/api/relationships/suggestions", relationshipHandler.Suggestions).Methods(http.MethodGet)

	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			logging.Logger().Error("health response write failed", zap.Error(err))
		}
	}).Methods(http.MethodGet)

	return requestLogger(cors(router))
}

type responseLogger struct {
	http.ResponseWriter
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

// requestLogger записывает входящие HTTP-запросы, ответы и необработанные panic
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &responseLogger{ResponseWriter: w, status: http.StatusOK}
		requestBody := readRequestBodyForLog(r)

		defer func() {
			if recovered := recover(); recovered != nil {
				logging.Logger().Error("panic recovered in HTTP handler",
					zap.Any("panic", recovered),
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
				zap.Int("status", response.status),
				zap.Duration("duration", time.Since(startedAt)),
			}
			if response.status >= http.StatusInternalServerError {
				logging.Logger().Error("HTTP request completed", fields...)
			} else {
				logging.Logger().Info("HTTP request completed", fields...)
			}
			logging.Logger().Debug("HTTP request details",
				zap.String("query", r.URL.RawQuery),
				zap.String("user_agent", r.UserAgent()),
			)
			if requestBody != "" {
				logging.Logger().Debug("HTTP request body",
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
func cors(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = os.Getenv("FRONTEND_URL")
	}
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8001"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (origin == allowedOrigin || allowedOrigin == "*") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
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
