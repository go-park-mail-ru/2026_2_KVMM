package profile

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	profileusecase "kvmm/internal/usecase/profile"
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
	session := helpers.SessionCookie(r)
	var req domain.UpdateProfileRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "")
		return
	}
	if err := helpers.ValidateUpdateProfileRequest(req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "")
		return
	}
	updatedProfile, err := h.useCase.Update(r.Context(), session, r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		switch err {
		case profileusecase.ErrInvalidCSRF:
			helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		case profileusecase.ErrUnauthorized:
			helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		case profileusecase.ErrConflict:
			helpers.WriteError(w, http.StatusConflict, "already_exists", "nickname, email or phone_number is already used")
		default:
			helpers.WriteError(w, http.StatusBadRequest, "invalid_request", "")
		}
		return
	}
	helpers.WriteJSON(w, http.StatusOK, updatedProfile)
}
