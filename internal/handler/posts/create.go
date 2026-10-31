package posts

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	postusecase "kvmm/internal/usecase/post"
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
	var req domain.CreatePostRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	post, err := h.useCase.Create(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, post)
}

func (h *PostHandler) writeError(w http.ResponseWriter, err error) {
	switch err {
	case postusecase.ErrUnauthorized:
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case postusecase.ErrInvalidCSRF:
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
	case postusecase.ErrForbidden:
		helpers.WriteError(w, http.StatusForbidden, "forbidden", "you can change only your own post")
	case postusecase.ErrInvalidPost:
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		helpers.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
