package auth

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Register регистрирует пользователя и создаёт сессию
// Register godoc
// @Summary Регистрация пользователя
// @Description Регистрирует пользователя и создаёт авторизованную сессию
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "Данные регистрации"
// @Success 201 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.RegisterRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateRegisterRequest(req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	profile, session, csrfToken, err := store.Register(req)
	if err != nil {
		if err == store.ErrConflict {
			helpers.WriteError(w, http.StatusConflict, "already_exists", "nickname, email or phone_number is already used")
			return
		}
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	setSessionCookie(w, session)
	helpers.WriteJSON(w, http.StatusCreated, domain.AuthResponse{Profile: profile, CSRFToken: csrfToken})
}
