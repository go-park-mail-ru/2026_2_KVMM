package session

import (
	"errors"

	"kvmm/internal/logging"
)

var (
	ErrUnauthorized = errors.New("authentication required") // сессия отсутствует или недействительна
	ErrInvalidCSRF  = errors.New("csrf validation failed")  // неверный CSRF-токен
)

// SessionReader — публичная часть репозитория сессий
// Её используют usecase, которым нужно только прочитать сессию или проверить CSRF
type SessionReader interface {
	Read(string) (int64, string, error)
	ValidateCSRF(string, string) error
}

// UseCase предоставляет общие операции проверки сессии другим usecase
type UseCase struct {
	reader SessionReader
}

// New создаёт общий usecase чтения и проверки сессий
func New(reader SessionReader) *UseCase {
	return &UseCase{reader: reader}
}

// Read возвращает пользователя и CSRF-токен действующей сессии
func (u *UseCase) Read(token string) (int64, string, error) {
	if token == "" {
		return 0, "", ErrUnauthorized
	}
	userID, csrfToken, err := u.reader.Read(token)
	if err != nil || userID <= 0 {
		if err != nil {
			logging.LogUseCaseError("session", "read session", err)
		}
		return 0, "", ErrUnauthorized
	}
	return userID, csrfToken, nil
}

// Authorize проверяет сессию и CSRF-токен защищённого запроса
func (u *UseCase) Authorize(token, csrfToken string) (int64, error) {
	userID, _, err := u.Read(token)
	if err != nil {
		return 0, err
	}
	if err := u.reader.ValidateCSRF(token, csrfToken); err != nil {
		return 0, ErrInvalidCSRF
	}
	return userID, nil
}
