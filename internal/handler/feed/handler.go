package feed

import feedusecase "kvmm/internal/usecase/feed"

type FeedHandler struct {
	useCase *feedusecase.UseCase
}

// NewFeedHandler создаёт обработчик запросов ленты
func NewFeedHandler(useCase *feedusecase.UseCase) *FeedHandler {
	return &FeedHandler{useCase: useCase}
}
