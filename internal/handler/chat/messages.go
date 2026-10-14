package chat

import (
	"net/http"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// Messages возвращает сообщения конкретного чата с пагинацией
// Messages godoc
// @Summary Получить сообщения чата
// @Description Возвращает сообщения указанного чата в хронологическом порядке
// @Tags chats
// @Security SessionCookie
// @Produce json
// @Param chat_id query int true "Идентификатор чата"
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
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// cursor, limit, err := helpers.Pagination(r, 10)
	// if err != nil {
	// 	helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
	// 	return
	// }

	// TODO: проверить участие пользователя в чате и получить его сообщения
}
