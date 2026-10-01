package handler

const sessionCookieName = "session_id"

type AuthHandler struct{}

// NewAuthHandler создаёт обработчик авторизации
func NewAuthHandler() *AuthHandler { return &AuthHandler{} }
