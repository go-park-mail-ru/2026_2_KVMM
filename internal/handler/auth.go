package handler

import "kvmm/internal/service"

const sessionCookieName = "session_id"

type AuthHandler struct{ auth *service.AuthService }

// NewAuthHandler создаёт обработчик авторизации.
func NewAuthHandler(auth *service.AuthService) *AuthHandler { return &AuthHandler{auth: auth} }
