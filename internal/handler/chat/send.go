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
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/chats/send [post]
func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	var req domain.SendMessageRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !helpers.IsUUID(req.ChatUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "chat_uid must be a valid UUID")
		return
	}
	if req.RepostFromUID != nil && !helpers.IsUUID(*req.RepostFromUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "repost_from must be a valid UUID")
		return
	}
	if req.ReplyFromUID != nil && !helpers.IsUUID(*req.ReplyFromUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "reply_from must be a valid UUID")
		return
	}

	message, err := h.useCase.SendByUID(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, message)
}
