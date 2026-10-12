package relationship

import (
	"net/http"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Requests возвращает входящие запросы на связи
// Requests godoc
// @Summary Получить входящие запросы на связи
// @Description Возвращает активные запросы на связи, отправленные текущему пользователю
// @Tags relationships
// @Security SessionCookie
// @Produce json
// @Param relationships_type query string false "Фильтр по виду связи" example(friendship)
// @Param offset query int false "Количество пропускаемых записей" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.RelationshipsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/relationships/requests [get]
func (h *FriendsHandler) Requests(w http.ResponseWriter, r *http.Request) {
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
	// cursor, limit, err := helpers.Pagination(r)

	// TODO
}
