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
	profile := domain.Profile{
		Nickname: strings.TrimSpace(req.Nickname), Email: req.Email, PhoneNumber: req.PhoneNumber,
		ProfileName: strings.TrimSpace(req.ProfileName), Surname: strings.TrimSpace(req.Surname),
		Patronymic: req.Patronymic, Gender: req.Gender, Bio: req.Bio, CreatedAt: now,
	}
	if req.Birthday != nil && *req.Birthday != "" {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return domain.Profile{}, "", "", err
		}
		profile.Birthday = &birthday
	}

	mu.Lock()
	defer mu.Unlock()
	for _, existing := range users {
		if strings.EqualFold(existing.Profile.Nickname, profile.Nickname) ||
			sameContact(existing.Profile.Email, profile.Email) ||
			sameContact(existing.Profile.PhoneNumber, profile.PhoneNumber) {
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
