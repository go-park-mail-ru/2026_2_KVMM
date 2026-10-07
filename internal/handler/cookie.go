package handler

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const sessionCookieName = "session_id"

// setSessionCookie устанавливает cookie
func setSessionCookie(w http.ResponseWriter, session string) {
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

// deleteSessionCookie удаляет cookie сессии пользователя
func deleteSessionCookie(w http.ResponseWriter) {
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
