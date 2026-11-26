package community

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"

	"github.com/gorilla/mux"
)

// Read возвращает сообщество по UUID
// @Summary Получить сообщество
// @Tags communities
// @Produce json
// @Param uid path string true "UUID сообщества"
// @Success 200 {object} domain.Community
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/communities/{uid} [get]
func (h *Handler) Read(w http.ResponseWriter, r *http.Request) {
	uid, ok := communityUID(w, r)
	if !ok {
		return
	}
	item, err := h.useCase.Read(r.Context(), uid)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, item)
}

// Update изменяет сообщество
// @Summary Изменить сообщество
// @Tags communities
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.UpdateCommunityRequest true "Новые значения"
// @Success 200 {object} domain.Community
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/communities/{uid} [patch]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	uid, ok := communityUID(w, r)
	if !ok {
		return
	}
	var req domain.UpdateCommunityRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	item, err := h.useCase.Update(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid, req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if h.search != nil {
		if err := h.search.IndexCommunity(r.Context(), item); err != nil {
			helpers.WriteInternalError(w, "community.update.search_index", err)
			return
		}
	}
	helpers.WriteJSON(w, http.StatusOK, item)
}

// Delete мягко удаляет сообщество владельцем
// @Summary Удалить сообщество
// @Tags communities
// @Security SessionCookie
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Сообщество удалено"
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/communities/{uid} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := communityUID(w, r)
	if !ok {
		return
	}
	if err := h.useCase.Delete(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid); err != nil {
		h.writeError(w, err)
		return
	}
	if h.search != nil {
		if err := h.search.DeleteDocument(r.Context(), "communities", uid); err != nil {
			helpers.WriteInternalError(w, "community.delete.search_index", err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func communityUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := mux.Vars(r)["uid"]
	if !helpers.IsUUID(uid) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
		return "", false
	}
	return uid, true
}
