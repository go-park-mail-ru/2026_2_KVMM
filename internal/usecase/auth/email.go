package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
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
	generatedPasswordLen = 32
	maxNicknameAttempts  = 20
)

var emailNicknameInvalidChars = regexp.MustCompile(`[^A-Za-z0-9]`)
var generatedPasswordAlphabet = []byte("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*_-+")

// RequestRegisterEmailCode проверяет свободный email и отправляет код регистрации
func (u *UseCase) RequestRegisterEmailCode(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	if _, err := u.profiles.ReadByLoginAuth(ctx, email); err == nil {
		// Не раскрываем, зарегистрирован ли email. Письмо не отправляем,
		// но резервируем адрес на время общего cooldown.
		u.sender.ReserveAuthCode(email)
		return nil
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

	password, err := randomPassword()
	if err != nil {
		return domain.AuthResponse{}, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthResponse{}, "", err
	}
	nickname, err := u.generatedEmailNickname(ctx, email)
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
			u.sender.ReserveAuthCode(email)
			return nil
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

// randomSecret создаёт hex-секрет для внутренних технических значений
func randomSecret() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// randomPassword создаёт сложный пароль, который не передаётся клиенту
func randomPassword() (string, error) {
	password := make([]byte, generatedPasswordLen)
	required := []byte("Aa1!")
	copy(password, required)

	for i := len(required); i < len(password); i++ {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(generatedPasswordAlphabet))))
		if err != nil {
			return "", err
		}
		password[i] = generatedPasswordAlphabet[value.Int64()]
	}

	for i := len(password) - 1; i > 0; i-- {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := int(value.Int64())
		password[i], password[j] = password[j], password[i]
	}
	return string(password), nil
}

// generatedEmailNickname создаёт уникальный nickname из локальной части email.
func (u *UseCase) generatedEmailNickname(ctx context.Context, email string) (string, error) {
	local := email
	if atPos := strings.IndexByte(email, '@'); atPos != -1 {
		local = email[:atPos]
	}

	nickname := emailNicknameInvalidChars.ReplaceAllString(local, "")
	if nickname == "" {
		return "", ErrConflict
	}
	startsWithLetter := (nickname[0] >= 'A' && nickname[0] <= 'Z') ||
		(nickname[0] >= 'a' && nickname[0] <= 'z')
	if !startsWithLetter {
		nickname = "user_" + nickname
	}
	for len(nickname) < 4 {
		nickname += nickname
	}
	if len(nickname) > 32 {
		nickname = nickname[:32]
	}

	free, err := u.isFreeNick(ctx, nickname)
	if err != nil {
		return "", err
	}
	if free {
		return nickname, nil
	}

	base := nickname
	if len(base) > 30 {
		base = base[:30]
	}
	for i := 0; i < maxNicknameAttempts; i++ {
		secret, err := randomSecret()
		if err != nil {
			return "", err
		}
		candidate := base + strings.ToLower(secret[:2])
		free, err := u.isFreeNick(ctx, candidate)
		if err != nil {
			return "", err
		}
		if free {
			return candidate, nil
		}
	}

	for i := 0; i < maxNicknameAttempts; i++ {
		secret, err := randomSecret()
		if err != nil {
			return "", err
		}
		candidate := "user_" + strings.ToLower(secret[:27])
		free, err := u.isFreeNick(ctx, candidate)
		if err != nil {
			return "", err
		}
		if free {
			return candidate, nil
		}
	}
	return "", ErrConflict
}

func (u *UseCase) isFreeNick(ctx context.Context, nickname string) (bool, error) {
	if _, err := u.profiles.ReadByLoginAuth(ctx, nickname); err == nil {
		return false, nil
	} else if !errors.Is(err, repository.ErrProfileNotFound) {
		return false, err
	}
	return true, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func emailCodeKey(email, purpose string, userID int64) string {
	return fmt.Sprintf("%s:%s:%d", purpose, normalizeEmail(email), userID)
}
