package posts

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Create создаёт публикацию от имени текущего пользователя
// Create godoc
// @Summary Создать публикацию
// @Description Создаёт публикацию текущего пользователя. Автор определяется по сессии
// @Tags posts
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CreatePostRequest true "Данные публикации"
// @Success 201 {object} domain.Post
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/create [post]
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
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

	var req domain.CreatePostRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// TODO: сохранить публикацию
}
