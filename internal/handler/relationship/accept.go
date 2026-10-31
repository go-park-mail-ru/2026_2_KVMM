package relationship

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Accept Принимает запрос на связь после проверки CSRF-токена
// Accept godoc
// @Summary Принять запрос на связь
// @Description Принимает запрос на связь указанному пользователю
// @Tags relationships
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.AcceptRelationshipRequest true "Данные для принятия запроса в друзья"
// @Success 200 {object} domain.ProfileRelationship "Принятая связь"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/relationships/accept [patch]
func (h *FriendsHandler) Accept(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPatch) {
		return
	}

	var req domain.AcceptRelationshipRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	relationship, err := h.useCase.Accept(
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
