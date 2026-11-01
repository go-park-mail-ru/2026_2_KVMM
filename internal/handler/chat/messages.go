package chat

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// Messages возвращает сообщения конкретного чата с пагинацией
// Messages godoc
// @Summary Получить сообщения чата
// @Description Возвращает сообщения указанного чата в хронологическом порядке
// @Tags chats
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Produce json
// @Param chat_id query int true "ID чата"
// @Param cursor query int false "Указатель на начальный элемент для сообщений" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.MessagesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/chats/messages [get]
func (h *ChatHandler) Messages(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	chatID, err := helpers.PositiveQueryInt64(r, "chat_id")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_chat_id", err.Error())
		return
	}
	cursor, limit, err := helpers.Pagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.Messages(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		chatID,
		int64(cursor),
		limit,
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
