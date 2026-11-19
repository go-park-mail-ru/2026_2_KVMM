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
// @Description Возвращает публичные данные. Можно передать uid или login
// @Tags profile
// @Produce json
// @Security SessionCookie
// @Param uid query string false "uid"
// @Param login query string false "Логин, email или номер телефона, UUID"
// @Success 200 {object} domain.Profile
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/profile/search [get]
func (h *ProfileHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("profile_id") != "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "profile_id is no longer accepted; use uid")
		return
	}
	if uid := strings.TrimSpace(query.Get("uid")); uid != "" {
		if !helpers.IsUUID(uid) {
			helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
			return
		}
		h.writeSearchResult(w, func() (domain.Profile, error) {
			return h.useCase.SearchByUUID(
				r.Context(),
				helpers.SessionCookie(r),
				uid,
			)
		})
		return
	}
	login := strings.TrimSpace(query.Get("login"))
	if login == "" {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "login is required")
		return
	}
	if _, err := strconv.ParseInt(login, 10, 64); err == nil {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "numeric profile IDs are not accepted; use uid")
		return
	}
	h.writeSearchResult(w, func() (domain.Profile, error) {
		return h.useCase.Search(
			r.Context(),
			helpers.SessionCookie(r),
			login,
		)
	})
}

// writeSearchResult отправляет результат публичного поиска профиля
func (h *ProfileHandler) writeSearchResult(w http.ResponseWriter, search func() (domain.Profile, error)) {
	profile, err := search()
	if err != nil {
		switch {
		case errors.Is(err, profileusecase.ErrUnauthorized):
			helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		case errors.Is(err, profileusecase.ErrProfileNotFound):
			helpers.WriteError(w, http.StatusNotFound, "profile_not_found", "profile not found")
		default:
			helpers.WriteInternalError(w, "profile.search", err)
		}
		return
	}
	helpers.WriteJSON(w, http.StatusOK, profile)
}
