package auth

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
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
	session := SessionCookie(r)
	profile, err := store.ProfileBySession(session)
	if err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	csrfToken, err := store.CSRFTokenBySession(session)
	if err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, domain.MeResponse{Profile: profile, CSRFToken: csrfToken})
}

// Logout завершает текущую сессию после проверки CSRF-токена.
// Logout godoc
// @Summary Выход из аккаунта
// @Description Завершает текущую сессию пользователя.
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
	session := SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := store.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}
	store.Logout(session)
	deleteSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// SessionCookie возвращает значение cookie сессии из запроса.
func SessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
