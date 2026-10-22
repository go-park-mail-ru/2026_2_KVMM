package posts

import (
	"net/http"
	"strconv"

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
// @Param post_id query int true "ID публикации"
// @Success 204 "Публикация удалена"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/delete [delete]
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodDelete) {
		return
	}
	postID, err := strconv.ParseInt(r.URL.Query().Get("post_id"), 10, 64)
	if err != nil || postID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "post_id must be a positive integer")
		return
	}
	if err := h.useCase.Delete(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), postID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
