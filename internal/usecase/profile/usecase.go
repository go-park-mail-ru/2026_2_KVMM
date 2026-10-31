package profile

import (
	"context"
	"strings"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized   = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF    = sessionusecase.ErrInvalidCSRF
	ErrConflict       = repository.ErrProfileConflict
	ErrAvatarLimit    = repository.ErrAvatarLimit
	ErrAvatarNotFound = repository.ErrAvatarNotFound
	ErrInvalidAvatar  = repository.ErrInvalidAvatar
)

type profileRepository interface {
	Update(context.Context, int64, repository.UpdateProfileParams) (domain.Profile, error)
}

type avatarRepository interface {
	Create(context.Context, int64, domain.UploadAvatarRequest) (domain.Avatar, error)
	List(context.Context, int64) (domain.AvatarsResponse, error)
	Delete(context.Context, int64, int64) error
}

// UseCase содержит бизнес-логику изменения профиля
type UseCase struct {
	profiles profileRepository
	avatars  avatarRepository
	sessions *sessionusecase.UseCase
}

// New создаёт usecase профиля
func New(
	profiles profileRepository,
	avatars avatarRepository,
	sessions *sessionusecase.UseCase,
) *UseCase {
	return &UseCase{profiles: profiles, avatars: avatars, sessions: sessions}
}

// Update проверяет сессию, хеширует новый пароль и изменяет переданные поля
func (u *UseCase) Update(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.UpdateProfileRequest,
) (domain.Profile, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Profile{}, err
	}

	params := repository.UpdateProfileParams{
		Nickname:    req.Nickname,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		ProfileName: req.ProfileName,
		Surname:     req.Surname,
		Patronymic:  req.Patronymic,
		Gender:      req.Gender,
		Bio:         req.Bio,
	}

	if req.Birthday != nil {
		value := strings.TrimSpace(*req.Birthday)
		if value != "" {
			birthday, parseErr := time.Parse("2006-01-02", value)
			if parseErr != nil {
				return domain.Profile{}, parseErr
			}
			params.Birthday = &birthday
		}
	}

	if req.Password != nil {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			return domain.Profile{}, hashErr
		}
		hashString := string(hash)
		params.PasswordHash = &hashString
	}

	return u.profiles.Update(ctx, userID, params)
}

// UploadAvatar загружает новую аватарку текущего пользователя
func (u *UseCase) UploadAvatar(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.UploadAvatarRequest,
) (domain.Avatar, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Avatar{}, err
	}
	return u.avatars.Create(ctx, userID, req)
}

// ListAvatars возвращает аватарки текущего пользователя от новой к старой
func (u *UseCase) ListAvatars(ctx context.Context, sessionToken string) (domain.AvatarsResponse, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.AvatarsResponse{}, err
	}
	return u.avatars.List(ctx, userID)
}

// DeleteAvatar удаляет выбранную аватарку текущего пользователя
func (u *UseCase) DeleteAvatar(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	avatarID int64,
) error {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	return u.avatars.Delete(ctx, userID, avatarID)
}
