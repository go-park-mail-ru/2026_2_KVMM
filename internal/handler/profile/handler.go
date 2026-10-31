package profile

import profileusecase "kvmm/internal/usecase/profile"

type ProfileHandler struct {
	useCase *profileusecase.UseCase
}

// NewProfileHandler создаёт обработчик профиля
func NewProfileHandler(useCase *profileusecase.UseCase) *ProfileHandler {
	return &ProfileHandler{useCase: useCase}
}
