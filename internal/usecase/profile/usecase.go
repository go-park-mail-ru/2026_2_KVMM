package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF  = sessionusecase.ErrInvalidCSRF
	ErrConflict     = errors.New("profile already exists")
)

type profileRepository interface {
	Update(context.Context, int64, repository.UpdateProfileParams) (domain.Profile, error)
}

// UseCase содержит бизнес-логику изменения профиля
type UseCase struct {
	profiles profileRepository
	sessions *sessionusecase.UseCase
}

// New создаёт usecase профиля
func New(profiles profileRepository, sessions *sessionusecase.UseCase) *UseCase {
	return &UseCase{profiles: profiles, sessions: sessions}
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
