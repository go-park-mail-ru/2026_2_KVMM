package helpers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"kvmm/internal/domain"
)

const sessionCookieName = "session_id"

// SessionCookie возвращает значение cookie сессии из запроса
func SessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetSessionCookie устанавливает cookie сессии
func SetSessionCookie(w http.ResponseWriter, session string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session,
		Path:     "/",
		MaxAge:   int((30 * 24 * time.Hour) / time.Second),
		HttpOnly: true,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		SameSite: cookieSameSite(),
	})
}

// DeleteSessionCookie удаляет cookie сессии
func DeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("COOKIE_SECURE") == "true",
		SameSite: cookieSameSite(),
	})
}

// cookieSameSite возвращает режим SameSite из переменной окружения
func cookieSameSite() http.SameSite {
	switch strings.ToLower(os.Getenv("COOKIE_SAMESITE")) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

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
