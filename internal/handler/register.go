package handler

import (
	"net/http"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/store"
)

// Register регистрирует пользователя и создаёт сессию.
// Register godoc
// @Summary Регистрация пользователя
// @Description Регистрирует пользователя и создаёт авторизованную сессию.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "Данные регистрации"
// @Success 201 {object} domain.AuthResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := validateRegisterRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	profile, session, csrfToken, err := h.auth.Register(req)
	if err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "already_exists", "nickname, email or phone_number is already used")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	setSessionCookie(w, session)
	writeJSON(w, http.StatusCreated, domain.AuthResponse{Profile: profile, CSRFToken: csrfToken})
}

// isEmptyContact проверяет, что optional-контакт отсутствует или пустой.
func isEmptyContact(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}
