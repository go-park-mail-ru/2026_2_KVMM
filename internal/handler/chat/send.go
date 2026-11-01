package chat

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Send отправляет текстовое сообщение в чат
// Send godoc
// @Summary Отправить сообщение
// @Description Отправляет текстовое сообщение в чат, в котором состоит текущий пользователь
// @Tags chats
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.SendMessageRequest true "Текст сообщения"
// @Success 201 {object} domain.Message
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/chats/send [post]
func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}
	var req domain.SendMessageRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	message, err := h.useCase.Send(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, message)
}
