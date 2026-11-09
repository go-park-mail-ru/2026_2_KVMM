package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	"kvmm/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

const (
	registerEmailPurpose = "register"
	loginEmailPurpose    = "login"
	changeEmailPurpose   = "change_email"
	emailCodeTTL         = 2 * time.Minute
)

// RequestRegisterEmailCode проверяет свободный email и отправляет код регистрации
func (u *UseCase) RequestRegisterEmailCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if _, err := u.profiles.ReadByLoginAuth(ctx, email); err == nil {
		return ErrConflict
	} else if !errors.Is(err, repository.ErrProfileNotFound) {
		logging.LogUseCaseErrorContext(ctx, "auth", "check registration email", err)
		return err
	}
	return u.sendEmailCode(ctx, email, registerEmailPurpose, 0)
}

// RegisterByEmail подтверждает код и регистрирует пользователя по email
func (u *UseCase) RegisterByEmail(ctx context.Context, req domain.RegisterByEmailRequest) (domain.AuthResponse, string, error) {
	email := normalizeEmail(req.Email)
	if !u.verifyEmailCode(email, registerEmailPurpose, 0, req.Code) {
		return domain.AuthResponse{}, "", ErrInvalidEmailCode
	}

	password, err := randomSecret()
	if err != nil {
		return domain.AuthResponse{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthResponse{}, "", err
	}
	nickname, err := generatedEmailNickname()
	if err != nil {
		return domain.AuthResponse{}, "", err
	}

	profile, err := u.profiles.Create(ctx, repository.CreateProfileParams{
		Nickname:     nickname,
		Email:        &email,
		PasswordHash: string(hash),
	})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "auth", "register by email", err)
		return domain.AuthResponse{}, "", err
	}
	return u.createAuthResponse(ctx, profile)
}

// RequestLoginEmailCode отправляет код входа существующему пользователю
func (u *UseCase) RequestLoginEmailCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	profile, err := u.profiles.ReadByLoginAuth(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			return ErrInvalidCredentials
		}
		logging.LogUseCaseErrorContext(ctx, "auth", "check login email", err)
		return err
	}
	return u.sendEmailCode(ctx, email, loginEmailPurpose, profile.Profile.ID)
}

// LoginByEmail подтверждает код и создаёт сессию пользователя
func (u *UseCase) LoginByEmail(ctx context.Context, req domain.LoginByEmailRequest) (domain.AuthResponse, string, error) {
	email := normalizeEmail(req.Email)
	profile, err := u.profiles.ReadByLoginAuth(ctx, email)
	if err != nil {
		return domain.AuthResponse{}, "", ErrInvalidCredentials
	}
	if !u.verifyEmailCode(email, loginEmailPurpose, profile.Profile.ID, req.Code) {
		return domain.AuthResponse{}, "", ErrInvalidCredentials
	}
	return u.createAuthResponse(ctx, profile.Profile)
}

// RequestEmailChange отправляет код на новый email текущего пользователя
func (u *UseCase) RequestEmailChange(ctx context.Context, sessionToken, csrfToken, email string) error {
	userID, err := u.reader.Authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	email = normalizeEmail(email)
	if profile, readErr := u.profiles.ReadByLoginAuth(ctx, email); readErr == nil && profile.Profile.ID != userID {
		return ErrConflict
	} else if readErr != nil && !errors.Is(readErr, repository.ErrProfileNotFound) {
		logging.LogUseCaseErrorContext(ctx, "auth", "check new email", readErr)
		return readErr
	}
	return u.sendEmailCode(ctx, email, changeEmailPurpose, userID)
}

// ConfirmEmailChange подтверждает новый email и сохраняет его в профиле
func (u *UseCase) ConfirmEmailChange(ctx context.Context, sessionToken, csrfToken string, req domain.ConfirmEmailRequest) (domain.Profile, error) {
	userID, err := u.reader.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Profile{}, err
	}
	email := normalizeEmail(req.Email)
	if !u.verifyEmailCode(email, changeEmailPurpose, userID, req.Code) {
		return domain.Profile{}, ErrInvalidEmailCode
	}
	profile, err := u.profiles.Update(ctx, userID, repository.UpdateProfileParams{Email: &email})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "auth", "confirm email change", err)
	}
	return profile, err
}

// sendEmailCode создаёт код, отправляет его и сохраняет на время проверки
func (u *UseCase) sendEmailCode(ctx context.Context, email, purpose string, userID int64) error {
	code, err := numericCode()
	if err != nil {
		return err
	}
	if err := u.sender.SendAuthCode(email, code); err != nil {
		logging.LogUseCaseErrorContext(ctx, "auth", "send email code", err)
		return fmt.Errorf("%w: %v", ErrEmailDelivery, err)
	}

	u.emailMu.Lock()
	u.codes[emailCodeKey(email, purpose, userID)] = emailCode{
		code:      code,
		expiresAt: time.Now().Add(emailCodeTTL),
	}
	u.emailMu.Unlock()
	return nil
}

// verifyEmailCode проверяет код, срок действия и удаляет использованный код
func (u *UseCase) verifyEmailCode(email, purpose string, userID int64, code string) bool {
	key := emailCodeKey(email, purpose, userID)
	u.emailMu.Lock()
	defer u.emailMu.Unlock()

	record, ok := u.codes[key]
	if !ok || time.Now().After(record.expiresAt) || subtle.ConstantTimeCompare([]byte(record.code), []byte(code)) != 1 {
		return false
	}
	delete(u.codes, key)
	return true
}

// numericCode создаёт шестизначный одноразовый код
func numericCode() (string, error) {
	data := make([]byte, 6)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	for i := range data {
		data[i] = '0' + data[i]%10
	}
	return string(data), nil
}

// randomSecret создаёт пароль, который не передаётся клиенту
func randomSecret() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// generatedEmailNickname создаёт технический уникальный nickname
func generatedEmailNickname() (string, error) {
	secret, err := randomSecret()
	if err != nil {
		return "", err
	}
	return "user_" + strings.ToLower(secret[:8]), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func emailCodeKey(email, purpose string, userID int64) string {
	return fmt.Sprintf("%s:%s:%d", purpose, normalizeEmail(email), userID)
}
