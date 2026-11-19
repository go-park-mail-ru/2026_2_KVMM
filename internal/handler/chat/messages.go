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
// @Produce json
// @Param chat_uid query string true "Внешний UUID чата"
// @Param cursor query string false "UUID последнего сообщения предыдущей страницы"
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.MessagesResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/chats/messages [get]
func (h *ChatHandler) Messages(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("chat_id") != "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "use chat_uid")
		return
	}
	chatUID, err := helpers.UUIDQuery(r, "chat_uid")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_chat_uid", err.Error())
		return
	}
	cursor, limit, err := helpers.UUIDPagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.MessagesByUID(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), chatUID, cursor, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
