package auth

import authusecase "kvmm/internal/usecase/auth"

type AuthHandler struct {
	useCase *authusecase.UseCase
}

// NewAuthHandler создаёт обработчик авторизации
func NewAuthHandler(useCase *authusecase.UseCase) *AuthHandler {
	return &AuthHandler{useCase: useCase}
}
