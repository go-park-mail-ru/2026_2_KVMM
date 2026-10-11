package friends

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Accept принимает входящий запрос в друзья
// Accept godoc
// @Summary Принять запрос в друзья
// @Description Принимает входящий запрос от указанного пользователя и устанавливает взаимную дружбу
// @Tags friends
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.AcceptFriendRequest true "Данные для принятия запроса в друзья"
// @Success 204 "Запрос принят"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/friends/accept [patch]
func (h *FriendsHandler) Accept(w http.ResponseWriter, r *http.Request) {
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

	var req domain.AcceptFriendRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// TODO
}
