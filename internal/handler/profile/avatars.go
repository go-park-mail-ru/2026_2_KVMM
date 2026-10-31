package profile

import (
	"errors"
	"net/http"

	"kvmm/internal/handler/helpers"
	profileusecase "kvmm/internal/usecase/profile"
)

// ListAvatars возвращает активные аватарки текущего пользователя
// ListAvatars godoc
// @Summary Получить аватарки профиля
// @Description Возвращает аватарки от самой новой к самой старой
// @Tags profile
// @Produce json
// @Security SessionCookie
// @Success 200 {object} domain.AvatarsResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/profile/avatars [get]
func (h *ProfileHandler) ListAvatars(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	response, err := h.useCase.ListAvatars(r.Context(), helpers.SessionCookie(r))
	if err != nil {
		h.writeAvatarError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}

func (h *ProfileHandler) writeAvatarError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profileusecase.ErrUnauthorized):
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, profileusecase.ErrInvalidCSRF):
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case errors.Is(err, profileusecase.ErrAvatarLimit):
		helpers.WriteError(w, http.StatusConflict, "avatar_limit", "maximum 10 active avatars are allowed")
	case errors.Is(err, profileusecase.ErrAvatarNotFound):
		helpers.WriteError(w, http.StatusNotFound, "avatar_not_found", "avatar not found")
	case errors.Is(err, profileusecase.ErrInvalidAvatar):
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar", "only JPEG, PNG, WEBP and GIF images are supported")
	default:
		helpers.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
