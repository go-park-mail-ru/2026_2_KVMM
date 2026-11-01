package auth

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	authusecase "kvmm/internal/usecase/auth"
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
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.LoginRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !helpers.PasswordPattern.MatchString(req.Password) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "password has invalid format")
		return
	}
	response, session, err := h.useCase.Login(r.Context(), req)
	if err != nil {
		if err == authusecase.ErrInvalidCredentials {
			helpers.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	helpers.SetSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusOK, response)
}
