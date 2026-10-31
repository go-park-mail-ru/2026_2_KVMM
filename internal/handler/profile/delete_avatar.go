package profile

import (
	"net/http"
	"strconv"

	"kvmm/internal/handler/helpers"
)

// DeleteAvatar удаляет аватарку текущего пользователя по идентификатору
// DeleteAvatar godoc
// @Summary Удалить аватарку
// @Description Удаляет выбранную аватарку текущего пользователя
// @Tags profile
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param id path int true "ID аватарки"
// @Success 204 "Аватарка удалена"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/profile/avatars/{id} [delete]
func (h *ProfileHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodDelete) {
		return
	}

	avatarID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || avatarID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar_id", "avatar id must be a positive integer")
		return
	}
	if err := h.useCase.DeleteAvatar(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		avatarID,
	); err != nil {
		h.writeAvatarError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
