package auth

import (
	"kvmm/internal/handler/helpers"
	authusecase "kvmm/internal/usecase/auth"
)

type AuthHandler struct {
	useCase        *authusecase.UseCase
	cookieSettings helpers.CookieSettings
}

// NewAuthHandler создаёт обработчик авторизации
func NewAuthHandler(useCase *authusecase.UseCase, settings ...helpers.CookieSettings) *AuthHandler {
	cookieSettings := helpers.CookieSettings{SameSite: "lax"}
	if len(settings) > 0 {
		cookieSettings = settings[0]
	}
	return &AuthHandler{useCase: useCase, cookieSettings: cookieSettings}
}
