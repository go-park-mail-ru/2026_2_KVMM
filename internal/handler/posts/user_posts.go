package posts

import (
	"net/http"

	// "kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// UserPosts возвращает публикации конкретного пользователя с пагинацией
// UserPosts godoc
// @Summary Получить публикации пользователя
// @Description Возвращает публикации указанного пользователя в обратном хронологическом порядке
// @Tags posts
// @Produce json
// @Param profile_id query int true "ID пользователя"
// @Param cursor query int false "Количество пропускаемых публикаций" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.UserPostsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/user [get]
func (h *PostHandler) UserPosts(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}

	// cursor, limit, err := helpers.Pagination(r, 10)
	// if err != nil {
	// 	helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
	// 	return
	// }
}
