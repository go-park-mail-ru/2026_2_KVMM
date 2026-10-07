package handler

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/store"
)

// Login авторизует пользователя по логину и паролю
// Login godoc
// @Summary Авторизация пользователя
// @Description Авторизует пользователя и создаёт сессию
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Данные авторизации"
// @Success 200 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !passwordPattern.MatchString(req.Password) {
		writeError(w, http.StatusBadRequest, "validation_error", "password has invalid format")
		return
	}
	profile, session, csrfToken, err := store.Login(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
		return
	}
	setSessionCookie(w, session)
	writeJSON(w, http.StatusOK, domain.AuthResponse{Profile: profile, CSRFToken: csrfToken})
}
