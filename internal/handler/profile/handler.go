package profile

import (
	"context"
	"kvmm/internal/domain"
	profileusecase "kvmm/internal/usecase/profile"
)

type ProfileHandler struct {
	useCase *profileusecase.UseCase
	search  interface {
		IndexProfile(context.Context, domain.Profile) error
	}
}

func NewProfileHandler(useCase *profileusecase.UseCase) *ProfileHandler {
	return &ProfileHandler{useCase: useCase}
}
func (h *ProfileHandler) SetSearchClient(s interface {
	IndexProfile(context.Context, domain.Profile) error
}) { h.search = s }
