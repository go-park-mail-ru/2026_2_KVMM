package chat

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// List возвращает список чатов текущего пользователя с пагинацией
// List godoc
// @Summary Получить список чатов
// @Description Возвращает чаты, в которых состоит текущий пользователь
// @Tags chats
// @Security SessionCookie
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Produce json
// @Param cursor query int false "Указатель на начальный элемент для чатов" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.ChatsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/chats/list [get]
func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	cursor, limit, err := helpers.Pagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.List(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		int64(cursor),
		limit,
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
