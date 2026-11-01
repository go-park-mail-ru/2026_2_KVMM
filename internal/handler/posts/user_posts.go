package posts

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// UserPosts возвращает публикации конкретного пользователя с пагинацией
// UserPosts godoc
// @Summary Получить публикации пользователя
// @Description Возвращает публикации указанного пользователя в обратном хронологическом порядке
// @Tags posts
// @Produce json
// @Param profile_id query int true "ID пользователя"
// @Param cursor query int false "Указатель на начальный элемент для публикаций" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.UserPostsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/user [get]
func (h *PostHandler) UserPosts(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}

	profileID, err := helpers.PositiveQueryInt64(r, "profile_id")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_profile_id", err.Error())
		return
	}
	cursor, limit, err := helpers.Pagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.UserPosts(r.Context(), profileID, int64(cursor), limit)
	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "profile or posts not found")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
