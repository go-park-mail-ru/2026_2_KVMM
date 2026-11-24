package community

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Create создаёт сообщество и назначает владельцем текущего пользователя
// @Summary Создать сообщество
// @Tags communities
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CreateCommunityRequest true "Данные сообщества"
// @Success 201 {object} domain.Community
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/communities [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateCommunityRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	item, err := h.useCase.Create(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, item)
}
