package relationship

import (
	"net/http"

	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/store"
)

// List возвращает список связей текущего пользователя по типу
// List godoc
// @Summary Получить список связей по типу
// @Description Возвращает связи текущего пользователя с фильтрацией по статусу и виду
// @Tags relationships
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string false "CSRF-токен, если требуется политикой приложения"
// @Param relationships_status query string false "Статус связи" Enums(request,approved,canceled)
// @Param relationships_type query string false "Вид связи" example(friendship)
// @Param cursor query int false "Указатель на начальный элемент для записей" default(0)
// @Param limit query int false "Размер страницы (максимум 10)" default(10) maximum(10)
// @Success 200 {object} domain.RelationshipsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/relationships/list [get]
func (h *FriendsHandler) List(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := store.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}

	// cursor, limit, err := helpers.Pagination(r, 10)
	// if err != nil {
	// 	helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
	// 	return
	// }

	// TODO
}

// TypesList возвращает список в связях какого виды учатсвыет пользователь
// TypesList godoc
// @Summary Получить список связей по типам
// @Description Возвращает виды связий в которых участвует пользователь
// @Tags relationships
// @Security SessionCookie
// @Accept json
// @Produce json
// @Param X-CSRF-Token header string false "CSRF-токен, если требуется политикой приложения"
// @Success 200 {object} domain.RelationshipsResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Router /api/relationships/typeslist [get]
func (h *FriendsHandler) TypesList(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodGet) {
		return
	}
	session := auth.SessionCookie(r)
	if _, err := store.ProfileBySession(session); err != nil {
		helpers.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := store.ValidateCSRF(session, r.Header.Get("X-CSRF-Token")); err != nil {
		helpers.WriteError(w, http.StatusForbidden, "csrf_failed", "valid X-CSRF-Token header is required")
		return
	}

	// TODO
}
