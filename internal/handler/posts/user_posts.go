package posts

import (
	"errors"
	"net/http"

	"kvmm/internal/handler/helpers"
	postusecase "kvmm/internal/usecase/post"
)

// UserPosts возвращает публикации конкретного пользователя с пагинацией
// UserPosts godoc
// @Summary Получить публикации пользователя
// @Description Возвращает публикации указанного пользователя в обратном хронологическом порядке
// @Tags posts
// @Produce json
// @Param profile_uid query string true "Внешний UUID пользователя"
// @Param cursor query string false "UUID последней публикации предыдущей страницы"
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.UserPostsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/posts/user [get]
func (h *PostHandler) UserPosts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("profile_id") != "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "profile_id is no longer accepted; use profile_uid")
		return
	}

	profileUID, err := helpers.UUIDQuery(r, "profile_uid")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_profile_uid", err.Error())
		return
	}
	cursor, limit, err := helpers.UUIDPagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.UserPostsByUID(r.Context(), profileUID, cursor, limit)
	if err != nil {
		if errors.Is(err, postusecase.ErrProfileNotFound) {
			helpers.WriteError(w, http.StatusNotFound, "not_found", "profile or posts not found")
		} else {
			helpers.WriteInternalError(w, "posts.user", err)
		}
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
