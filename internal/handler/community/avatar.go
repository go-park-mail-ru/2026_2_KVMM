package community

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

// UploadAvatar загружает новую аватарку сообщества.
// @Summary Загрузить аватарку сообщества
// @Description Принимает PNG, JPEG, WebP или GIF размером не более 10 МБ. Загрузить изображение могут владелец и администратор сообщества.
// @Tags communities
// @Security SessionCookie
// @Accept multipart/form-data
// @Produce json
// @Param uid path string true "UUID сообщества"
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param avatar formData file true "Файл аватарки"
// @Success 201 {object} domain.Avatar
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 404 {object} domain.ErrorResponse
// @Failure 500 {object} domain.ErrorResponse
// @Router /api/communities/{uid}/avatar [post]
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)["uid"]
	if !helpers.IsUUID(uid) {
		helpers.WriteError(w, http.StatusBadRequest, "validation_error", "uid must be a valid UUID")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(domain.MaxAvatarSize)+1024*1024)
	file, _, err := r.FormFile("avatar")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar", "multipart field avatar is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(domain.MaxAvatarSize)+1))
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar file could not be read")
		return
	}
	if len(data) == 0 || len(data) > domain.MaxAvatarSize {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar must be between 1 byte and 10 MB")
		return
	}
	avatar, err := h.useCase.UploadAvatar(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		uid,
		domain.UploadAvatarRequest{Data: data},
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, avatar)
}
