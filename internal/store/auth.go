package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"kvmm/internal/domain"
)

const (
	EmailCodeRegister = "register"
	EmailCodeLogin    = "login"
	emailCodeTTL      = 2 * time.Minute
)

// ValidateCSRF проверяет CSRF-токен сессии
func ValidateCSRF(sessionToken, token string) error {
	expected, err := CSRFTokenBySession(sessionToken)
	if err != nil || token == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

// CSRFTokenBySession возвращает CSRF токен сессии
func CSRFTokenBySession(token string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()
	session, ok := sessions[token]
	if !ok {
		return "", ErrNotFound
	}
	return session.CSRFToken, nil
}

// ProfileBySession возвращает профиль пользователя по сессии
func ProfileBySession(token string) (domain.Profile, error) {
	mu.RLock()
	defer mu.RUnlock()
	session, ok := sessions[token]
	if !ok {
		return domain.Profile{}, ErrNotFound
	}
	user, ok := users[session.UserID]
	if !ok {
		return domain.Profile{}, ErrNotFound
	}
	return user.Profile, nil
}

// randomToken генерирует случайный CSRF токен для сессии
func randomToken() string {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}
	return hex.EncodeToString(data)
}

// Logout удаляет сессию
func Logout(token string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessions, token)
}

// CreateEmailCode создаёт и сохраняет шестизначный код подтверждения email
func CreateEmailCode(email, purpose string) (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", value.Int64()+100000)
	key := emailCodeKey(email, purpose)

	mu.Lock()
	emailCodes[key] = EmailCode{Code: code, ExpiresAt: time.Now().UTC().Add(emailCodeTTL)}
	mu.Unlock()

	return code, nil
}

// CancelEmailCode удаляет код, если письмо с ним не удалось отправить
func CancelEmailCode(email, purpose string) {
	mu.Lock()
	defer mu.Unlock()
	delete(emailCodes, emailCodeKey(email, purpose))
}

// ConsumeEmailCode проверяет код и удаляет его после успешного использования.
func ConsumeEmailCode(email, purpose, code string) error {
	key := emailCodeKey(email, purpose)

	mu.Lock()
	defer mu.Unlock()
	stored, ok := emailCodes[key]
	if !ok || time.Now().UTC().After(stored.ExpiresAt) ||
		subtle.ConstantTimeCompare([]byte(stored.Code), []byte(strings.TrimSpace(code))) != 1 {
		return ErrInvalidEmailCode
	}
	delete(emailCodes, key)
	return nil
}

// emailCodeKey создаёт ключ для отдельного кода регистрации или входа.
func emailCodeKey(email, purpose string) string {
	return purpose + ":" + strings.ToLower(strings.TrimSpace(email))
}
