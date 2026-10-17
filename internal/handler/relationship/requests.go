package relationship

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// Requests возвращает входящие запросы на связи
// Requests godoc
// @Summary Получить входящие запросы на связи
// @Description Возвращает активные запросы на связи, отправленные текущему пользователю
// @Tags relationships
// @Security SessionCookie
// @Produce json
// @Param relationships_type query string false "Фильтр по виду связи" example(friendship)
// @Param cursor query int false "Указатель на начальный элемент для записей" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.RelationshipsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/relationships/requests [get]
func (h *FriendsHandler) Requests(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}

	// cursor, limit, err := helpers.Pagination(r, 10)
	// if err != nil {
	// 	helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
	// 	return
	// }

}
