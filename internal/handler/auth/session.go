package auth

import (
	"net/http"

	"kvmm/internal/handler/helpers"
	authusecase "kvmm/internal/usecase/auth"
)

// Me возвращает профиль текущего пользователя и CSRF-токен его сессии
// Me godoc
// @Summary Текущий пользователь
// @Description Возвращает пользователя по авторизованной сессии
// @Tags auth
// @Produce json
// @Security SessionCookie
// @Success 200 {object} domain.MeResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	session := helpers.SessionCookie(r)
	response, err := h.useCase.Current(r.Context(), session)
	if err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}

// Logout завершает текущую сессию после проверки CSRF-токена
// Logout godoc
// @Summary Выход из аккаунта
// @Description Завершает текущую сессию пользователя
// @Tags auth
// @Security SessionCookie
// @Success 204
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Router /api/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	session := helpers.SessionCookie(r)
	if err := h.useCase.Logout(session, r.Header.Get("X-CSRF-Token")); err != nil {
		switch err {
		case authusecase.ErrInvalidCSRF:
			helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		default:
			helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		}
		return
	}
	helpers.DeleteSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
