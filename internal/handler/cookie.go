package handler

import (
	"net/http"
	"os"
	"time"
)

// setSessionCookie устанавливает долгоживущую HTTP cookie сессии.
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

// deleteSessionCookie удаляет cookie сессии пользователя.
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

// cookieSameSite возвращает режим SameSite из переменной окружения.
func cookieSameSite() http.SameSite {
	if os.Getenv("COOKIE_SAMESITE") == "none" {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}
