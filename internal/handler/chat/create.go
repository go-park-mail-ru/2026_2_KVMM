package chat

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Create создаёт чат с указанным пользователем
// Create godoc
// @Summary Создать чат
// @Description Создаёт чат между текущим пользователем и указанным пользователем
// @Tags chats
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CreateChatRequest true "Пользователь для создания чата"
// @Success 201 {object} domain.Chat
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/chats/create [post]
func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.CreateChatRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	chat, err := h.useCase.Create(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, chat)
}
