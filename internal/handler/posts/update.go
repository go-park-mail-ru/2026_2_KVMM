package posts

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
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
	var req domain.UpdatePostRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	post, err := h.useCase.Update(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, post)
}
