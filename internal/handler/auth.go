package handler

type AuthHandler struct{}

// NewAuthHandler создаёт обработчик авторизации
func NewAuthHandler() *AuthHandler { return &AuthHandler{} }
