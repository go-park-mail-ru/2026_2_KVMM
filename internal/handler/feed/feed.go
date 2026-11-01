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
// @Param cursor query int false "Указатель на начальный элемент для постов" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Produce json
// @Success 200 {object} domain.FeedResponse
// @Router /api/feed [get]
func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	cursor, limit, err := helpers.Pagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.List(r.Context(), int64(cursor), limit)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
