package httpapi

import (
	"bytes"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kvmm/internal/handler"
	"kvmm/internal/store"
)

// NewRouter создаёт маршрутизатор API на основе стандартного http.ServeMux
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	authHandler := handler.NewAuthHandler()
	feedHandler := handler.NewFeedHandler()
	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/me", authHandler.Me)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/posts", feedHandler.List)
	mux.HandleFunc("/api/media/", serveMedia)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return cors(mux)
}

// serveMedia отдаёт встроенный файл публикации с Сontent-Type
func serveMedia(w http.ResponseWriter, r *http.Request) {
	if !handler.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/media/")
	data, err := readMediaFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// readMediaFile возвращает встроенный файл публикации по безопасному имени
func readMediaFile(name string) ([]byte, error) {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) {
		return nil, store.ErrNotFound
	}
	return store.EmbeddedMediaFiles.ReadFile("media/" + name)
}
