package httpapi

import (
	"net/http"
	"os"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/chat"
	"kvmm/internal/handler/feed"
	"kvmm/internal/handler/posts"
	"kvmm/internal/handler/profile"
	"kvmm/internal/handler/relationship"

	"github.com/gorilla/mux"
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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}).Methods(http.MethodGet)

	return cors(router)
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
