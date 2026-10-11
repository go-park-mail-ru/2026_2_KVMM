package helpers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/go-playground/validator/v10"
)

const (
	maxBodySize = 16 << 10
	minAge      = 14
	maxAge      = 120
)

var validate = validator.New()

var (
	nicknamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.]*$`)
	emailPattern    = regexp.MustCompile(`^[A-Za-z0-9_%+-]+(?:\.[A-Za-z0-9_%+-]+)*@(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z]{2,63}$`)
	phonePattern    = regexp.MustCompile(`^\+[1-9][0-9]{9,14}$`)
	namePattern     = regexp.MustCompile(`^[\p{L}\p{M}]+(?:[ '’-][\p{L}\p{M}]+)*$`)
	PasswordPattern = regexp.MustCompile(`^[\x21-\x7E]+$`)
	letterPattern   = regexp.MustCompile(`[A-Za-z]`)
	digitPattern    = regexp.MustCompile(`[0-9]`)
)

// validateRegisterRequest проверяет правила регистрации, которые нельзя описать только тегами
func ValidateRegisterRequest(req domain.RegisterRequest) error {
	if !nicknamePattern.MatchString(req.Nickname) {
		return fmt.Errorf("nickname may contain only latin letters, digits, _ and .")
	}
	if !PasswordPattern.MatchString(req.Password) || !letterPattern.MatchString(req.Password) || !digitPattern.MatchString(req.Password) {
		return fmt.Errorf("password must contain latin letters and digits without spaces")
	}
	return nil
}

// ValidateUpdateProfileRequest проверяет поля, которые пришли для изменения профиля
func ValidateUpdateProfileRequest(req domain.UpdateProfileRequest) error {
	if req.Nickname == nil && req.Email == nil && req.PhoneNumber == nil && req.Password == nil &&
		req.ConfirmPassword == nil && req.ProfileName == nil && req.Surname == nil && req.Patronymic == nil &&
		req.Gender == nil && req.Birthday == nil && req.Bio == nil {
		return fmt.Errorf("at least one field is required")
	}
	if req.Nickname != nil && !nicknamePattern.MatchString(strings.TrimSpace(*req.Nickname)) {
		return fmt.Errorf("nickname may contain only latin letters, digits, _ and .")
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		value := strings.TrimSpace(*req.Email)
		if !emailPattern.MatchString(value) || strings.Index(value, "@") > 64 {
			return fmt.Errorf("email has invalid format")
		}
	}
	if req.PhoneNumber != nil && strings.TrimSpace(*req.PhoneNumber) != "" && !phonePattern.MatchString(strings.TrimSpace(*req.PhoneNumber)) {
		return fmt.Errorf("phone_number has invalid format")
	}
	if req.ProfileName != nil && !namePattern.MatchString(strings.TrimSpace(*req.ProfileName)) {
		return fmt.Errorf("profile_name may contain only letters, spaces, apostrophes and hyphens")
	}
	if req.Surname != nil && !namePattern.MatchString(strings.TrimSpace(*req.Surname)) {
		return fmt.Errorf("surname may contain only letters, spaces, apostrophes and hyphens")
	}
	if req.Patronymic != nil && strings.TrimSpace(*req.Patronymic) != "" && !namePattern.MatchString(strings.TrimSpace(*req.Patronymic)) {
		return fmt.Errorf("patronymic may contain only letters, spaces, apostrophes and hyphens")
	}
	if req.Gender != nil && *req.Gender != domain.GenderMale && *req.Gender != domain.GenderFemale {
		return fmt.Errorf("gender is invalid")
	}
	if req.Birthday != nil && strings.TrimSpace(*req.Birthday) != "" {
		birthday, err := time.Parse("2006-01-02", strings.TrimSpace(*req.Birthday))
		if err != nil {
			return fmt.Errorf("birthday must have format YYYY-MM-DD")
		}
		now := time.Now().UTC()
		if birthday.AddDate(minAge, 0, 0).After(now) || !birthday.AddDate(maxAge+1, 0, 0).After(now) {
			return fmt.Errorf("age must be between 14 and 120 years")
		}
	}
	if req.Bio != nil && len([]rune(*req.Bio)) > 256 {
		return fmt.Errorf("bio must be at most 256 characters")
	}
	if req.Password != nil {
		if !PasswordPattern.MatchString(*req.Password) || !letterPattern.MatchString(*req.Password) || !digitPattern.MatchString(*req.Password) {
			return fmt.Errorf("password must contain latin letters and digits without spaces")
		}
		if req.ConfirmPassword == nil || *req.ConfirmPassword != *req.Password {
			return fmt.Errorf("password confirmation does not match")
		}
	} else if req.ConfirmPassword != nil {
		return fmt.Errorf("password is required with confirm_password")
	}
	return nil
}
