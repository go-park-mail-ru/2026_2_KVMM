package session

import "errors"

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrInvalidCSRF  = errors.New("csrf validation failed")
)

// SessionReader - публичная часть репозитория сессий
type SessionReader interface {
	Read(string) (int64, string, error)
	ValidateCSRF(string, string) error
}

type UseCase struct {
	reader SessionReader
}

// New создаёт общий usecase чтения сессий
func New(reader SessionReader) *UseCase {
	return &UseCase{reader: reader}
}

// Read возвращает пользователя и CSRF-токен сессии
func (u *UseCase) Read(token string) (int64, string, error) {
	if token == "" {
		return 0, "", ErrUnauthorized
	}
	userID, csrfToken, err := u.reader.Read(token)
	if err != nil || userID <= 0 {
		return 0, "", ErrUnauthorized
	}
	return userID, csrfToken, nil
}

// Authorize проверяет сессию и CSRF-токен
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
