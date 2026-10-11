package helpers

import (
	"encoding/json"
	"net/http"

	"kvmm/internal/domain"
)

// writeJSON сериализует значение и отправляет JSON-ответ с указанным статусом
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError отправляет клиенту ошибку в едином формате
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, domain.ErrorResponse{Code: code, Message: message})
}

// decodeJSON читает JSON из тела запроса с ограничением размера и проверяет его структуру
func DecodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return validate.Struct(target)
}

// methodAllowed проверяет HTTP-метод и возвращает ошибку для неподдерживаемого метода
func MethodAllowed(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed")
	return false
}
