package community

import (
	"net/http"

	"kvmm/internal/handler/helpers"
)

// List возвращает публичный список сообществ
// @Summary Получить список сообществ
// @Tags communities
// @Produce json
// @Param offset query int false "Смещение" default(0)
// @Param limit query int false "Размер страницы" default(10) maximum(50)
// @Success 200 {object} domain.CommunityPage
// @Router /api/communities [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := pagination(r)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	page, err := h.useCase.List(r.Context(), offset, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, page)
}

// Subscriptions возвращает подписки текущего пользователя
// @Summary Получить список подписок
// @Tags communities
// @Security SessionCookie
// @Produce json
// @Param offset query int false "offset" default(0)
// @Param limit query int false "limit" default(10) maximum(50)
// @Success 200 {object} domain.CommunityPage
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/communities/subscriptions [get]
func (h *Handler) Subscriptions(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := pagination(r)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	page, err := h.useCase.Subscriptions(r.Context(), helpers.SessionCookie(r), offset, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, page)
}
