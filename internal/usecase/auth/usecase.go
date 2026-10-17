package auth

import (
	"context"
	"errors"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCSRF        = sessionusecase.ErrInvalidCSRF
	ErrConflict           = errors.New("profile already exists")
)

type profileRepository interface {
	Create(context.Context, repository.CreateProfileParams) (domain.Profile, error)
	ReadByID(context.Context, int64) (domain.Profile, error)
	ReadByLoginAuth(context.Context, string) (repository.AuthProfile, error)
}

type sessionRepository interface {
	sessionusecase.SessionReader
	Create(int64) (string, string, error)
	Delete(string)
}

// UseCase содержит сценарии регистрации, входа и работы с текущей сессией
type UseCase struct {
	profiles profileRepository
	sessions sessionRepository
	reader   *sessionusecase.UseCase
}

// New создаёт usecase авторизации
func New(profiles profileRepository, sessions sessionRepository) *UseCase {
	return &UseCase{
		profiles: profiles,
		sessions: sessions,
		reader:   sessionusecase.New(sessions),
	}
}

// Register регистрирует пользователя, хеширует пароль и создаёт сессию
func (u *UseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthResponse, string, error) {
	if existing, err := u.profiles.ReadByLoginAuth(ctx, req.Nickname); err == nil && existing.Profile.ID > 0 {
		return domain.AuthResponse{}, "", ErrConflict
	} else if err != nil && !errors.Is(err, repository.ErrProfileNotFound) {
		return domain.AuthResponse{}, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthResponse{}, "", err
	}

	profile, err := u.profiles.Create(ctx, repository.CreateProfileParams{
		Nickname:     strings.TrimSpace(req.Nickname),
		Email:        nil,
		PasswordHash: string(hash),
	})
	if err != nil {
		return domain.AuthResponse{}, "", err
	}

	return u.createAuthResponse(profile)
}

// Login проверяет логин и пароль и создаёт сессию
func (u *UseCase) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthResponse, string, error) {
	found, err := u.profiles.ReadByLoginAuth(ctx, strings.TrimSpace(req.Login))
	if err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			return domain.AuthResponse{}, "", ErrInvalidCredentials
		}
		return domain.AuthResponse{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(found.PasswordHash), []byte(req.Password)) != nil {
		return domain.AuthResponse{}, "", ErrInvalidCredentials
	}
	return u.createAuthResponse(found.Profile)
}

// Current возвращает профиль и CSRF-токен текущей сессии
func (u *UseCase) Current(ctx context.Context, sessionToken string) (domain.MeResponse, error) {
	userID, csrfToken, err := u.reader.Read(sessionToken)
	if err != nil {
		return domain.MeResponse{}, err
	}
	profile, err := u.profiles.ReadByID(ctx, userID)
	if err != nil {
		return domain.MeResponse{}, err
	}
	return domain.MeResponse{Profile: profile, CSRFToken: csrfToken}, nil
}

// Logout проверяет CSRF-токен и завершает текущую сессию
func (u *UseCase) Logout(sessionToken, csrfToken string) error {
	if _, err := u.reader.Authorize(sessionToken, csrfToken); err != nil {
		return err
	}
	u.sessions.Delete(sessionToken)
	return nil
}

func (u *UseCase) createAuthResponse(profile domain.Profile) (domain.AuthResponse, string, error) {
	sessionToken, csrfToken, err := u.sessions.Create(profile.ID)
	if err != nil {
		return domain.AuthResponse{}, "", err
	}
	return domain.AuthResponse{Profile: profile, CSRFToken: csrfToken}, sessionToken, nil
}
