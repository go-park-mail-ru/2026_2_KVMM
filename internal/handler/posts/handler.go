package posts

import postusecase "kvmm/internal/usecase/post"

type PostHandler struct {
	useCase *postusecase.UseCase
}

// NewProfileHandler создаёт обработчик профиля
func NewPostHandler(useCase *postusecase.UseCase) *PostHandler {
	return &PostHandler{useCase: useCase}
}
