package friends

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// List возвращает список друзей текущего пользователя
// List godoc
// @Summary Получить список друзей
// @Description Возвращает список пользователей, с которыми у текущего пользователя установлена дружба
// @Tags friends
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string false "CSRF-токен, если требуется политикой приложения"
// @Param cursor query int false "Количество пропускаемых аккаунтов" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.RelationshipsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/friends/list [get]
func (h *FriendsHandler) List(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
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

	var req domain.ListFriendsRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// TODO
}
