package community

import communityusecase "kvmm/internal/usecase/community"

type Handler struct {
	useCase *communityusecase.CommunityUseCase
}

func NewHandler(useCase *communityusecase.CommunityUseCase) *Handler {
	return &Handler{useCase: useCase}
}
