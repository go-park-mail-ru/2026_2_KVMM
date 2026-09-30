package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"kvmm/internal/domain"
)

var validate = validator.New()

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

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода.
func methodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}
