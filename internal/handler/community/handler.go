package community

import (
	"context"
	"kvmm/internal/domain"
	communityusecase "kvmm/internal/usecase/community"
)

type Handler struct {
	useCase *communityusecase.CommunityUseCase
	search  interface {
		IndexCommunity(context.Context, domain.Community) error
		DeleteDocument(context.Context, string, string) error
	}
}

func NewHandler(useCase *communityusecase.CommunityUseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) SetSearchClient(s interface {
	IndexCommunity(context.Context, domain.Community) error
	DeleteDocument(context.Context, string, string) error
}) {
	h.search = s
}
