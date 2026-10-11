package friends

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Delete удаляет пользователя из друзей или отменяет запрос
// Delete godoc
// @Summary Удалить друга или отменить запрос
// @Description Удаляет дружбу с указанным пользователем либо отменяет исходящий запрос в друзья
// @Tags friends
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.DeleteFriendRequest true "Данные для удаления дружбы или отмены запроса"
// @Success 204
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/friends/delete [delete]
func (h *FriendsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodDelete) {
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

	var req domain.DeleteFriendRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// TODO
}
