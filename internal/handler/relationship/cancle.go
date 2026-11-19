package relationship

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Cancle Отменяет входящую связь или удаляет созданую после проверки CSRF-токена
// Cancle godoc
// @Summary Отменить связь
// @Description Отменяет входящую связь или удаляет созданую
// @Tags relationships
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CancleRelationshipRequest true "Данные для отправки запроса в друзья"
// @Success 200 {object} domain.ProfileRelationship "Обновленная связь"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/relationships/cancle [delete]
func (h *FriendsHandler) Cancle(w http.ResponseWriter, r *http.Request) {
	var req domain.CancleRelationshipRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if !helpers.IsUUID(req.ProfileUID) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "profile_uid must be a valid UUID")
		return
	}
	relationship, err := h.useCase.CancelByUID(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		req,
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, relationship)
}
