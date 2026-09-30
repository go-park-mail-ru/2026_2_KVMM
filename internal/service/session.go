package service

import (
	"crypto/subtle"

	"kvmm/internal/domain"
)

// newSession создаёт идентификатор сессии и связанный с ним CSRF-токен.
func (s *AuthService) newSession(userID int64) (string, string) {
	session := randomToken()
	csrfToken := randomToken()
	s.store.SaveSession(session, userID, csrfToken)
	return session, csrfToken
}

// ProfileBySession возвращает профиль пользователя по идентификатору сессии.
func (s *AuthService) ProfileBySession(session string) (domain.Profile, error) {
	id, err := s.store.SessionUserID(session)
	if err != nil {
		return domain.Profile{}, err
	}
	user, err := s.store.UserByID(id)
	if err != nil {
		return domain.Profile{}, err
	}
	return user.Profile, nil
}

// CSRFTokenBySession возвращает CSRF-токен авторизованной сессии.
func (s *AuthService) CSRFTokenBySession(session string) (string, error) {
	return s.store.SessionCSRFToken(session)
}

// ValidateCSRF проверяет CSRF-токен защищённого запроса.
func (s *AuthService) ValidateCSRF(session, token string) error {
	expected, err := s.CSRFTokenBySession(session)
	if err != nil || token == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

// Logout удаляет сессию пользователя из хранилища.
func (s *AuthService) Logout(session string) { s.store.DeleteSession(session) }
