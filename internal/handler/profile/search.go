package profile

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	profileusecase "kvmm/internal/usecase/profile"
)

// Search ищет публичный профиль по ID, логину, email или номеру телефона
// Search godoc
// @Summary Найти пользователя
// @Description Возвращает публичные данные. Можно передать profile_id или login
// @Tags profile
// @Produce json
// @Param profile_id query int false "ID пользователя" minimum(1)
// @Param login query string false "Логин, email или номер телефона"
// @Success 200 {object} domain.Profile
// @Failure 400 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/profile/search [get]
func (h *ProfileHandler) Search(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}

	query := r.URL.Query()
	if profileID := strings.TrimSpace(query.Get("profile_id")); profileID != "" {
		id, err := strconv.ParseInt(profileID, 10, 64)
		if err != nil || id <= 0 {
			helpers.WriteError(w, http.StatusBadRequest, "validation_error", "profile_id must be a positive integer")
			return
		}
		h.writeSearchResult(w, func() (domain.Profile, error) {
			return h.useCase.SearchByID(r.Context(), id)
		})
		return
	}

	login := strings.TrimSpace(query.Get("login"))
	if login == "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "login is required")
		return
	}
	h.writeSearchResult(w, func() (domain.Profile, error) {
		return h.useCase.Search(r.Context(), login)
	})
}

// writeSearchResult отправляет результат публичного поиска профиля
func (h *ProfileHandler) writeSearchResult(w http.ResponseWriter, search func() (domain.Profile, error)) {
	profile, err := search()
	if err != nil {
		switch {
		case errors.Is(err, profileusecase.ErrProfileNotFound):
			helpers.WriteError(w, http.StatusNotFound, "profile_not_found", "profile not found")
		default:
			helpers.WriteInternalError(w, "profile.search", err)
		}
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profile)
}
