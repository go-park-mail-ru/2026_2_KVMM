package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized    = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF     = sessionusecase.ErrInvalidCSRF
	ErrProfileNotFound = repository.ErrProfileNotFound
	ErrConflict        = repository.ErrProfileConflict
	ErrEmailConfirm    = errors.New("email change requires confirmation")
	ErrAvatarLimit     = repository.ErrAvatarLimit
	ErrAvatarNotFound  = repository.ErrAvatarNotFound
	ErrInvalidAvatar   = repository.ErrInvalidAvatar
)

type profileRepository interface {
	ReadByID(context.Context, int64) (domain.Profile, error)
	ReadByLogin(context.Context, string) (domain.Profile, error)
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

// Search возвращает публичный профиль пользователя по логину, email или телефону
func (u *UseCase) Search(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	login string,
) (domain.Profile, error) {
	if _, err := u.sessions.Authorize(sessionToken, csrfToken); err != nil {
		return domain.Profile{}, err
	}
	profile, err := u.profiles.ReadByLogin(ctx, strings.TrimSpace(login))
	if err != nil {
		logging.LogUseCaseError("profile", "search profile", err)
	}
	return profile, err
}

// SearchByID возвращает публичный профиль пользователя по ID
func (u *UseCase) SearchByID(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	id int64,
) (domain.Profile, error) {
	if _, err := u.sessions.Authorize(sessionToken, csrfToken); err != nil {
		return domain.Profile{}, err
	}
	profile, err := u.profiles.ReadByID(ctx, id)
	if err != nil {
		logging.LogUseCaseError("profile", "search profile by id", err)
	}
	return profile, err
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
	if req.Email != nil {
		return domain.Profile{}, ErrEmailConfirm
	}

	params := repository.UpdateProfileParams{
		Nickname:    req.Nickname,
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
			logging.LogUseCaseError("profile", "hash password", hashErr)
			return domain.Profile{}, hashErr
		}
		hashString := string(hash)
		params.PasswordHash = &hashString
	}

	profile, err := u.profiles.Update(ctx, userID, params)
	if err != nil {
		logging.LogUseCaseError("profile", "update profile", err)
	}
	return profile, err
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
	avatar, err := u.avatars.Create(ctx, userID, req)
	if err != nil {
		logging.LogUseCaseError("profile", "create avatar", err)
	}
	return avatar, err
}

// ListAvatars возвращает аватарки текущего пользователя от новой к старой
func (u *UseCase) ListAvatars(ctx context.Context, sessionToken string) (domain.AvatarsResponse, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.AvatarsResponse{}, err
	}
	response, err := u.avatars.List(ctx, userID)
	if err != nil {
		logging.LogUseCaseError("profile", "list avatars", err)
	}
	return response, err
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
	err = u.avatars.Delete(ctx, userID, avatarID)
	if err != nil {
		logging.LogUseCaseError("profile", "delete avatar", err)
	}
	return err
}
