package store

import (
	"strings"
	"time"

	"kvmm/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// UpdateProfileBySession изменяет профиль пользователя, связанного с сессией
func UpdateProfileBySession(sessionToken string, req domain.UpdateProfileRequest) (domain.Profile, error) {
	var passwordHash []byte
	var err error
	if req.Password != nil {
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return domain.Profile{}, err
		}
	}

	mu.Lock()
	defer mu.Unlock()
	session, ok := sessions[sessionToken]
	if !ok {
		return domain.Profile{}, ErrNotFound
	}
	user, ok := users[session.UserID]
	if !ok {
		return domain.Profile{}, ErrNotFound
	}

	profile := user.Profile
	if req.Nickname != nil {
		profile.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.Email != nil {
		profile.Email = optionalString(*req.Email)
	}
	if req.PhoneNumber != nil {
		profile.PhoneNumber = optionalString(*req.PhoneNumber)
	}
	if req.ProfileName != nil {
		profile.ProfileName = optionalString(*req.ProfileName)
	}
	if req.Surname != nil {
		profile.Surname = optionalString(*req.Surname)
	}
	if req.Patronymic != nil {
		profile.Patronymic = optionalString(*req.Patronymic)
	}
	if req.Gender != nil {
		gender := *req.Gender
		profile.Gender = &gender
	}
	if req.Birthday != nil {
		value := strings.TrimSpace(*req.Birthday)
		if value == "" {
			profile.Birthday = nil
		} else {
			birthday, parseErr := time.Parse("2006-01-02", value)
			if parseErr != nil {
				return domain.Profile{}, parseErr
			}
			profile.Birthday = &birthday
		}
	}
	if req.Bio != nil {
		profile.Bio = optionalString(*req.Bio)
	}
	for id, existing := range users {
		if id == profile.ID {
			continue
		}
		if (profile.Nickname != "" && strings.EqualFold(existing.Profile.Nickname, profile.Nickname)) ||
			sameContact(existing.Profile.Email, profile.Email) ||
			sameContact(existing.Profile.PhoneNumber, profile.PhoneNumber) {
			return domain.Profile{}, ErrConflict
		}
	}

	now := time.Now().UTC()
	profile.UpdatedAt = &now
	user.Profile = profile
	if req.Password != nil {
		user.PasswordHash = passwordHash
	}
	users[profile.ID] = user
	return profile, nil
}

// optionalString превращает пустое значение в nil для необязательных полей
func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
