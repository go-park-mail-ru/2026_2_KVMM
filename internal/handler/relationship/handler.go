package relationship

import (
	relusecase "kvmm/internal/usecase/relationship"
)

type FriendsHandler struct {
	useCase *relusecase.UseCase
}

// NewRelationshipHandler создаёт обработчик связей
func NewRelationshipHandler(useCase *relusecase.UseCase) *FriendsHandler {
	return &FriendsHandler{useCase: useCase}
}
