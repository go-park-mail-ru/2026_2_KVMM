package posts

import (
	"net/http"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
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
// @Param request query int true "ID публикации"
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
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := store.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}

	// TODO: проверить автора публикации и выполнить soft delete
}
