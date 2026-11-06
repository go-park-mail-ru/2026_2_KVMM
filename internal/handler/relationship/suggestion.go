package relationship

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// Suggestions возвращает список профилей, которым текущий пользователь может имень связь
// Suggestions godoc
// @Summary Получить список профилей возможных связей получателя по типу
// @Description Возвращает список профилей, которым текущий пользователь может имень связь.
// @Tags relationships
// @Security SessionCookie
// @Produce json
// @Param cursor query int false "Указатель на начальный элемент для записей" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.RelationshipSuggestionsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/relationships/suggestions [get]
func (h *FriendsHandler) Suggestions(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}

	cursor, limit, err := helpers.Pagination(r, 10)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.Suggestions(r.Context(), helpers.SessionCookie(r), int64(cursor), limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
