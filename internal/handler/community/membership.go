package community

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"

	"github.com/gorilla/mux"
)

// Subscribe оформляет подписку текущего пользователя
// @Summary Подписаться на сообщество
// @Tags communities
// @Security SessionCookie
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Подписка оформлена"
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/communities/{uid}/subscribe [post]
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	uid, ok := communityUID(w, r)
	if !ok {
		return
	}
	if err := h.useCase.Subscribe(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Unsubscribe отменяет подписку текущего пользователя
// @Summary Отписаться от сообщества
// @Tags communities
// @Security SessionCookie
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Подписка отменена"
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/communities/{uid}/subscribe [delete]
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	uid, ok := communityUID(w, r)
	if !ok {
		return
	}
	if err := h.useCase.Unsubscribe(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AssignAdmin назначает администратора сообщества
// @Summary Назначить администратора
// @Tags communities
// @Security SessionCookie
// @Accept json
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.AssignCommunityAdminRequest true "UUID участника"
// @Success 204 "Администратор назначен"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Router /api/communities/{uid}/admins [post]
func (h *Handler) AssignAdmin(w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)["uid"]
	if !helpers.IsUUID(uid) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
		return
	}
	var req domain.AssignCommunityAdminRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !helpers.IsUUID(req.ProfileUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "profile_uid must be a valid UUID")
		return
	}
	if err := h.useCase.AssignAdmin(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid, req.ProfileUID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
