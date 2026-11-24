package posts

import (
	"net/http"

	"kvmm/internal/handler/helpers"

	"github.com/gorilla/mux"
)

// Like ставит лайк посту текущего пользователя
// @Summary Поставить лайк посту
// @Tags posts
// @Security SessionCookie
// @Param uid path string true "UUID публикации"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Лайк установлен"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/{uid}/like [put]
func (h *PostHandler) Like(w http.ResponseWriter, r *http.Request) {
	uid, ok := postUID(w, r)
	if !ok {
		return
	}
	if err := h.useCase.Like(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Unlike снимает лайк текущего пользователя
// @Summary Снять лайк с поста
// @Tags posts
// @Security SessionCookie
// @Param uid path string true "UUID публикации"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Success 204 "Лайк снят"
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/{uid}/like [delete]
func (h *PostHandler) Unlike(w http.ResponseWriter, r *http.Request) {
	uid, ok := postUID(w, r)
	if !ok {
		return
	}
	if err := h.useCase.Unlike(r.Context(), helpers.SessionCookie(r), r.Header.Get("X-CSRF-Token"), uid); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Likers возвращает страницу друзей, поставивших лайк публикации
// @Summary Получить друзей, лайкнувших публикацию
// @Tags posts
// @Security SessionCookie
// @Produce json
// @Param uid path string true "UUID публикации"
// @Param cursor query string false "UUID профиля из предыдущей страницы"
// @Param limit query int false "Размер страницы (максимум 50)" default(10) maximum(50)
// @Success 200 {object} domain.PostLikersResponse
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Router /api/posts/{uid}/likes [get]
func (h *PostHandler) Likers(w http.ResponseWriter, r *http.Request) {
	uid, ok := postUID(w, r)
	if !ok {
		return
	}
	cursor, limit, err := helpers.UUIDPagination(r, 50)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return
	}
	response, err := h.useCase.PostLikers(r.Context(), helpers.SessionCookie(r), uid, cursor, limit)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}

func postUID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := mux.Vars(r)["uid"]
	if !helpers.IsUUID(uid) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
		return "", false
	}
	return uid, true
}
