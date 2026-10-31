package relationship

import (
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// Create отправляет запрос на произвольную связь после проверки CSRF-токена
// Create godoc
// @Summary Отправить запрос на связь
// @Description Отправляет указанному пользователю запрос на связь выбранного вида
// @Tags relationships
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param request body domain.CreateRelationshipRequest true "Данные для создания связи"
// @Success 201 {object} domain.ProfileRelationship "Созданная связь"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/relationships/create [post]
func (h *FriendsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}

	var req domain.CreateRelationshipRequest
	if err := helpers.DecodeJSON(w, r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
}
