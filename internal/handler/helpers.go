package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	passwordPattern = regexp.MustCompile(`^[\x21-\x7E]+$`)
	letterPattern   = regexp.MustCompile(`[A-Za-z]`)
	digitPattern    = regexp.MustCompile(`[0-9]`)
)

// writeJSON сериализует значение и отправляет JSON-ответ с указанным статусом
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError отправляет клиенту ошибку в едином формате
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, domain.ErrorResponse{Code: code, Message: message})
}

// decodeJSON читает JSON из тела запроса с ограничением размера и проверяет его структуру
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return validate.Struct(target)
}

// validateRegisterRequest проверяет правила регистрации, которые нельзя описать только тегами
func validateRegisterRequest(req domain.RegisterRequest) error {
	if isEmptyContact(req.Email) && isEmptyContact(req.PhoneNumber) {
		return fmt.Errorf("email or phone_number is required")
	}
	if !nicknamePattern.MatchString(req.Nickname) {
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
	if !namePattern.MatchString(req.ProfileName) {
		return fmt.Errorf("profile_name may contain only letters, spaces, apostrophes and hyphens")
	}
	if !namePattern.MatchString(req.Surname) {
		return fmt.Errorf("surname may contain only letters, spaces, apostrophes and hyphens")
	}
	if req.Patronymic != nil && strings.TrimSpace(*req.Patronymic) != "" && !namePattern.MatchString(*req.Patronymic) {
		return fmt.Errorf("patronymic may contain only letters, spaces, apostrophes and hyphens")
	}
	if !passwordPattern.MatchString(req.Password) || !letterPattern.MatchString(req.Password) || !digitPattern.MatchString(req.Password) {
		return fmt.Errorf("password must contain latin letters and digits without spaces")
	}
	birthday, err := time.Parse("2006-01-02", strings.TrimSpace(*req.Birthday))
	if err != nil {
		return fmt.Errorf("birthday must have format YYYY-MM-DD")
	}
	now := time.Now().UTC()
	if birthday.AddDate(minAge, 0, 0).After(now) || !birthday.AddDate(maxAge+1, 0, 0).After(now) {
		return fmt.Errorf("age must be between 14 and 120 years")
	}
	return nil
}

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода
func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}

// isEmptyContact проверяет, что optional-контакт отсутствует или пустой
func isEmptyContact(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}
