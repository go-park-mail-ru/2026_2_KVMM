package chat

import (
	"net/http"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// List возвращает список чатов текущего пользователя с пагинацией
// List godoc
// @Summary Получить список чатов
// @Description Возвращает чаты, в которых состоит текущий пользователь
// @Tags chats
// @Security SessionCookie
// @Produce json
// @Param offset query int false "Количество пропускаемых чатов" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.ChatsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/chats/list [get]
func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// TODO: получить список чатов с offset/limit
}
