package relationship

import (
	"context"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF  = sessionusecase.ErrInvalidCSRF
	ErrConflict     = repository.ErrRelationshipConflict // у пары уже есть активная связь
	ErrNotFound     = repository.ErrRelationshipNotFound // связь или профиль не найдены
	ErrInvalid      = repository.ErrInvalidRelationship  // некорректные данные
)

type relationshipRepository interface {
	Create(context.Context, int64, domain.CreateRelationshipRequest) (domain.ProfileRelationship, error)
	List(context.Context, int64, domain.ListRelationshipsRequest) (domain.RelationshipsResponse, error)
	Requests(context.Context, int64, string, int64, int) (domain.RelationshipsResponse, error)
	Types(context.Context, int64) (domain.RelationshipTypes, error)
	Suggestions(context.Context, int64, int64, int) (domain.RelationshipSuggestionsResponse, error)
	Accept(context.Context, int64, domain.AcceptRelationshipRequest) (domain.ProfileRelationship, error)
	Cancel(context.Context, int64, domain.CancleRelationshipRequest) (domain.ProfileRelationship, error)
}

// UseCase содержит сценарии работы со связями пользователей
type UseCase struct {
	relationships relationshipRepository
	sessions      *sessionusecase.UseCase
}

// New создаёт usecase связей
func New(
	relationships relationshipRepository,
	sessions *sessionusecase.UseCase,
) *UseCase {
	return &UseCase{relationships: relationships, sessions: sessions}
}
