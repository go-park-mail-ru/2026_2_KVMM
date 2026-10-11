package store

import (
	"strings"
	"time"

	"kvmm/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// Register создаёт пользователя и сессию
func Register(req domain.RegisterRequest) (domain.Profile, string, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.Profile{}, "", "", err
	}
	now := time.Now().UTC()
	profile := domain.Profile{Nickname: strings.TrimSpace(req.Nickname), CreatedAt: now}

	mu.Lock()
	defer mu.Unlock()
	for _, existing := range users {
		if profile.Nickname != "" && strings.EqualFold(existing.Profile.Nickname, profile.Nickname) {
			return domain.Profile{}, "", "", ErrConflict
		}
	}
	profile.ID = nextID
	nextID++
	users[profile.ID] = User{Profile: profile, PasswordHash: hash}
	sessionToken := randomToken()
	csrfToken := randomToken()
	sessions[sessionToken] = Session{UserID: profile.ID, CSRFToken: csrfToken}
	return profile, sessionToken, csrfToken, nil
}

// sameContact сравнивает две строки без учёта регистра
func sameContact(left, right *string) bool {
	return left != nil && right != nil && strings.EqualFold(*left, *right)
}
