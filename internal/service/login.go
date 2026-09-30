package service

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
	"kvmm/internal/domain"
)

// Login проверяет учётные данные и создаёт авторизованную сессию.
func (s *AuthService) Login(req domain.LoginRequest) (domain.Profile, string, string, error) {
	user, err := s.store.FindUser(strings.TrimSpace(req.Login))
	if err != nil || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(req.Password)) != nil {
		return domain.Profile{}, "", "", ErrInvalidCredentials
	}
	session, csrfToken := s.newSession(user.Profile.ID)
	return user.Profile, session, csrfToken, nil
}
