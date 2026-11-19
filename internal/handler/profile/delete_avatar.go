package profile

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// DeleteAvatar удаляет аватарку текущего пользователя по id
// DeleteAvatar godoc
// @Summary Удалить аватарку
// @Description Удаляет выбранную аватарку текущего пользователя
// @Tags profile
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param uid path string true "uid"
// @Success 204 "Аватарка удалена"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/profile/avatars/{uid} [delete]
func (h *ProfileHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	avatarUID := r.PathValue("uid")
	if !helpers.IsUUID(avatarUID) {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar_id", "avatar id must be a UUID")
		return
	}
	if err := h.useCase.DeleteAvatarByUID(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		avatarUID,
	); err != nil {
		h.writeAvatarError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
