package posts

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Update изменяет содержание собственной публикации
// Update godoc
// @Summary Редактировать свою публикацию
// @Description Изменяет публикации текущего пользователя
// @Tags posts
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.UpdatePostRequest true "Новое содержание публикации"
// @Success 200 {object} domain.Post
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/update [patch]
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPatch) {
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

	var req domain.UpdatePostRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// TODO: проверить автора публикации и сохранить новый текст
}
