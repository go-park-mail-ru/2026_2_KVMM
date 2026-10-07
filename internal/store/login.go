package store

import (
	"strings"

	"kvmm/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// Login проверяет учётные данные и создаёт сессию
func Login(req domain.LoginRequest) (domain.Profile, string, string, error) {
	mu.Lock()
	defer mu.Unlock()
	login := strings.TrimSpace(req.Login)
	var found User
	for _, candidate := range users {
		if strings.EqualFold(candidate.Profile.Nickname, login) ||
			equalString(candidate.Profile.Email, login) ||
			equalString(candidate.Profile.PhoneNumber, login) {
			found = candidate
			break
		}
	}
	if found.Profile.ID == 0 || bcrypt.CompareHashAndPassword(found.PasswordHash, []byte(req.Password)) != nil {
		return domain.Profile{}, "", "", ErrInvalidCredentials
	}
	sessionToken := randomToken()
	csrfToken := randomToken()
	sessions[sessionToken] = Session{UserID: found.Profile.ID, CSRFToken: csrfToken}
	return found.Profile, sessionToken, csrfToken, nil
}

// equalString сравнивает строку value с target без учёта регистра
func equalString(value *string, target string) bool {
	return value != nil && strings.EqualFold(*value, target)
}
