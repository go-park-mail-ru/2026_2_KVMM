package feed

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// List возвращает последовательность публикаций ленты
// List godoc
// @Summary Лента публикаций
// @Description Возвращает посты в обратном хронологическом порядке
// @Tags feed
// @Param cursor query string false "UUID последнего поста предыдущей страницы"
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Produce json
// @Success 200 {object} domain.FeedResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/feed [get]
func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	cursor, limit, err := helpers.UUIDPagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.ListByUID(r.Context(), cursor, limit)
	if err != nil {
		helpers.WriteInternalError(w, "feed.list", err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
