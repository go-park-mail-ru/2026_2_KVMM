package auth

import (
	"context"
	"kvmm/internal/domain"
	"net/http"

	"kvmm/internal/handler/helpers"
	authusecase "kvmm/internal/usecase/auth"

	"github.com/gorilla/mux"
)

type AuthHandler struct {
	useCase *authusecase.UseCase
	search  interface {
		IndexProfile(context.Context, domain.Profile) error
	}
	cookieSettings helpers.CookieSettings
}

// NewRouter создаёт HTTP-маршруты, которыми владеет Auth service
func NewRouter(handler *AuthHandler) http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/api/auth/register", handler.Register).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/register/email/send/code", handler.RequestRegisterCode).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/register/email/code", handler.RegisterByEmail).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/login", handler.Login).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/login/email/send/code", handler.RequestLoginCode).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/login/email/code", handler.LoginByEmail).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/me", handler.Me).Methods(http.MethodGet)
	router.HandleFunc("/api/auth/logout", handler.Logout).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/email/change", handler.RequestEmailChange).Methods(http.MethodPost)
	router.HandleFunc("/api/auth/email/confirm", handler.ConfirmEmail).Methods(http.MethodPost)
	return router
}

// NewAuthHandler создаёт обработчик авторизации
func NewAuthHandler(useCase *authusecase.UseCase, settings ...helpers.CookieSettings) *AuthHandler {
	cookieSettings := helpers.CookieSettings{SameSite: "lax"}
	if len(settings) > 0 {
		cookieSettings = settings[0]
	}
	return &AuthHandler{useCase: useCase, cookieSettings: cookieSettings}
}

func (h *AuthHandler) SetSearchClient(s interface {
	IndexProfile(context.Context, domain.Profile) error
}) {
	h.search = s
}
func (h *AuthHandler) index(ctx context.Context, p domain.Profile) error {
	if h.search != nil {
		return h.search.IndexProfile(ctx, p)
	}
	return nil
}
