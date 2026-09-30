package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]*$`)
var namePattern = regexp.MustCompile(`^[\p{L}]+(?:-[\p{L}]+)*$`)

// writeJSON сериализует значение и отправляет JSON-ответ с указанным статусом.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError отправляет клиенту ошибку в едином формате API.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, domain.ErrorResponse{Code: code, Message: message})
}

// decodeJSON читает JSON из тела запроса и проверяет его структуру.
func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return validate.Struct(target)
}

// validateRegisterRequest проверяет правила регистрации, которые нельзя описать только тегами.
func validateRegisterRequest(req domain.RegisterRequest) error {
	if isEmptyContact(req.Email) && isEmptyContact(req.PhoneNumber) {
		return fmt.Errorf("email or phone_number is required")
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		value := strings.TrimSpace(*req.Email)
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value {
			return fmt.Errorf("email has invalid format")
		}
	}
	if req.PhoneNumber != nil && strings.TrimSpace(*req.PhoneNumber) != "" && !phonePattern.MatchString(strings.TrimSpace(*req.PhoneNumber)) {
		return fmt.Errorf("phone_number has invalid format")
	}
	if !namePattern.MatchString(req.ProfileName) {
		return fmt.Errorf("profile_name may contain only letters and hyphens")
	}
	if !namePattern.MatchString(req.Surname) {
		return fmt.Errorf("surname may contain only letters and hyphens")
	}
	if req.Patronymic != nil && strings.TrimSpace(*req.Patronymic) != "" && !namePattern.MatchString(*req.Patronymic) {
		return fmt.Errorf("patronymic may contain only letters and hyphens")
	}
	if req.Birthday != nil && strings.TrimSpace(*req.Birthday) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*req.Birthday)); err != nil {
			return fmt.Errorf("birthday must have format YYYY-MM-DD")
		}
	}
	return nil
}

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода.
func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}
