package posts

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// Delete удаляет публикацию текущего пользователя
// Delete godoc
// @Summary Удалить свою публикацию
// @Description Помечает публикацию текущего пользователя удалённой
// @Tags posts
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param post_uid query string true "Внешний UUID публикации"
// @Success 204 "Публикация удалена"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/posts/delete [delete]
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("post_id") != "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "post_id is no longer accepted; use post_uid")
		return
	}
	postUID, err := helpers.UUIDQuery(r, "post_uid")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if err := h.useCase.DeleteByUID(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), postUID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteCommunity удаляет публикацию сообщества.
// @Summary Удалить публикацию сообщества
// @Tags community-posts
// @Security SessionCookie
// @Param community_uid query string true "UUID сообщества"
// @Param post_uid query string true "UUID публикации"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Публикация удалена"
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/community-posts/delete [delete]
func (h *PostHandler) DeleteCommunity(w http.ResponseWriter, r *http.Request) {
	communityUID, postUID := r.URL.Query().Get("community_uid"), r.URL.Query().Get("post_uid")
	if !helpers.IsUUID(communityUID) || !helpers.IsUUID(postUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "community_uid and post_uid must be valid UUIDs")
		return
	}
	if err := h.useCase.DeleteCommunityPost(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), communityUID, postUID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
