package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"kvmm/internal/domain"
	"kvmm/internal/store"
)

type FeedHandler struct{}

// NewFeedHandler создаёт обработчик запросов ленты
func NewFeedHandler() *FeedHandler { return &FeedHandler{} }

// List возвращает последовательность публикаций ленты
// List godoc
// @Summary Лента публикаций
// @Description Возвращает посты в обратном хронологическом порядке
// @Tags posts
// @Param cursor query int false "Количество пропускаемых постов" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Produce json
// @Success 200 {object} domain.FeedResponse
// @Router /api/posts [get]
func (h *FeedHandler) List(w http.ResponseWriter, r *http.Request) {
	if !MethodAllowed(w, r, http.MethodGet) {
		return
	}
	cursor, limit, err := feedPagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	posts, hasMore, nextCursor := store.ListPosts(cursor, limit)
	response := domain.FeedResponse{Posts: posts, Cursor: cursor, Limit: limit, HasMore: hasMore}
	if hasMore {
		response.NextCursor = nextCursor
	}
	writeJSON(w, http.StatusOK, response)
}

// feedPagination читает cursor и размер страницы из query-параметров
func feedPagination(r *http.Request) (int, int, error) {
	cursor := 0
	limit := 10
	query := r.URL.Query()
	if value := query.Get("cursor"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return 0, 0, fmt.Errorf("cursor must be a non-negative integer")
		}
		cursor = parsed
	}
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 10 {
			return 0, 0, fmt.Errorf("limit must be between 1 and 10")
		}
		limit = parsed
	}
	return cursor, limit, nil
}
