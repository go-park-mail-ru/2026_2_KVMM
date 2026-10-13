package store

import (
	"strings"
	"time"

	"kvmm/internal/domain"

	mathrand "math/rand"

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

// EmailExists проверяет, зарегистрирован ли пользователь с таким email
func EmailExists(email string) bool {
	email = strings.TrimSpace(email)

	mu.RLock()
	defer mu.RUnlock()
	for _, user := range users {
		if sameContact(user.Profile.Email, &email) {
			return true
		}
	}
	return false
}

// RegisterByEmail регистрирует пользователя после проверки кода email
// Пароль создаётся случайно и не передаётся клиенту: дальнейший вход выполняется кодом либо сброс пароля
func RegisterByEmail(req domain.RegisterByEmailRequest) (domain.Profile, string, string, error) {
	if err := ConsumeEmailCode(req.Email, EmailCodeRegister, req.Code); err != nil {
		return domain.Profile{}, "", "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(randomToken()), bcrypt.DefaultCost)
	if err != nil {
		return domain.Profile{}, "", "", err
	}
	now := time.Now().UTC()
	email := strings.TrimSpace(req.Email)

	parts := strings.SplitN(email, "@", 2)
	nickname := parts[0]

	// TODO отвалидитовать никнейм

	profile := domain.Profile{
		Email:     &email,
		Nickname:  nickname,
		CreatedAt: now,
	}

	for {
		exists := false
		for _, existing := range users {
			if profile.Nickname != "" &&
				strings.EqualFold(existing.Profile.Nickname, profile.Nickname) {
				exists = true
				break
			}
		}
		if !exists {
			break
		}
		profile.Nickname += randomSuffix(4)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, existing := range users {
		if sameContact(existing.Profile.Email, profile.Email) {
			return domain.Profile{}, "", "", ErrConflict
		}
	}
	profile.ID = nextID
	nextID++
	users[profile.ID] = User{Profile: profile, PasswordHash: passwordHash}
	sessionToken := randomToken()
	csrfToken := randomToken()
	sessions[sessionToken] = Session{UserID: profile.ID, CSRFToken: csrfToken}
	return profile, sessionToken, csrfToken, nil
}

// sameContact сравнивает две строки без учёта регистра
func sameContact(left, right *string) bool {
	return left != nil && right != nil && strings.EqualFold(*left, *right)
}

func randomSuffix(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

	result := make([]byte, length)
	for i := range result {
		result[i] = chars[mathrand.Intn(len(chars))]
	}

	return string(result)
}
