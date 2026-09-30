package handler

import (
	"net/http"

	"kvmm/internal/domain"
)

// Me возвращает профиль текущего пользователя и CSRF-токен его сессии.
// Me godoc
// @Summary Текущий пользователь
// @Description Возвращает пользователя по авторизованной сессии.
// @Tags auth
// @Produce json
// @Security SessionCookie
// @Success 200 {object} domain.MeResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	session := sessionCookie(r)
	profile, err := h.auth.ProfileBySession(session)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	csrfToken, err := h.auth.CSRFTokenBySession(session)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, domain.MeResponse{Profile: profile, CSRFToken: csrfToken})
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
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	session := sessionCookie(r)
	if _, err := h.auth.ProfileBySession(session); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := h.auth.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		writeError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}
	h.auth.Logout(session)
	deleteSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// sessionCookie возвращает значение cookie сессии из запроса.
func sessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
