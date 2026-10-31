package profile

import (
	"io"
	"net/http"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
)

const avatarFormField = "avatar"

// UploadAvatar загружает новую аватарку текущего пользователя
// UploadAvatar godoc
// @Summary Загрузить аватарку
// @Description Загружает изображение аватарки. Максимум — 10 активных аватарок
// @Tags profile
// @Security SessionCookie
// @Accept multipart/form-data
// @Produce json
// @Param X-CSRF-Token header string true "CSRF-токен"
// @Param avatar formData file true "Файл аватарки"
// @Success 201 {object} domain.Avatar
// @Failure 400 {object} domain.ErrorResponse
// @Failure 401 {object} domain.ErrorResponse
// @Failure 403 {object} domain.ErrorResponse
// @Failure 409 {object} domain.ErrorResponse
// @Router /api/profile/avatar [post]
func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if !helpers.MethodAllowed(w, r, http.MethodPost) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(domain.MaxAvatarSize)+1024*1024)
	file, _, err := r.FormFile(avatarFormField)
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
	if len(data) > domain.MaxAvatarSize {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_avatar", "avatar file is too large")
		return
	}

	avatar, err := h.useCase.UploadAvatar(
		r.Context(),
		helpers.SessionCookie(r),
		r.Header.Get("X-CSRF-Token"),
		domain.UploadAvatarRequest{Data: data},
	)
	if err != nil {
		h.writeAvatarError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, avatar)
}
