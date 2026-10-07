package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"

	"kvmm/internal/domain"
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
