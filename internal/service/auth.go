package service

import (
	"errors"

	"kvmm/internal/store"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidCSRF = errors.New("invalid csrf token")

type AuthService struct{ store *store.Memory }

// NewAuthService создаёт сервис авторизации на основе in-memory хранилища.
func NewAuthService(s *store.Memory) *AuthService { return &AuthService{store: s} }
