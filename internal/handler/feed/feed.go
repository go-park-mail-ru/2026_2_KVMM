package feed

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
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
	posts, hasMore, nextCursor := store.ListPosts(cursor, limit)
	response := domain.FeedResponse{Posts: posts, Cursor: cursor, Limit: limit, HasMore: hasMore}
	if hasMore {
		response.NextCursor = nextCursor
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
