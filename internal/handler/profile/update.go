package profile

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Update изменяет профиль текущего пользователя
// Update godoc
// @Summary Изменяет профиль пользователя
// @Description Изменяет профиль существующего пользователя
// @Tags profile
// @Accept json
// @Produce json
// @Param request body domain.UpdateProfileRequest true "Изменяемые поля"
// @Security SessionCookie
// @Success 200 {object} domain.Profile
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/profile/update [patch]
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPatch) {
		return
	}
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := store.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}
	var req domain.UpdateProfileRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := helpers.ValidateUpdateProfileRequest(req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	profile, err := store.UpdateProfileBySession(session, req)
	if err != nil {
		switch err {
		case store.ErrConflict:
			helpers.WriteError(w, http.StatusConflict, "already_exists", "nickname, email or phone_number is already used")
		default:
			helpers.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profile)
}
