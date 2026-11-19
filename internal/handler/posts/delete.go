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
