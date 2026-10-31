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

// Create отправляет запрос на связь выбранного вида
func (u *UseCase) Create(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CreateRelationshipRequest,
) (domain.ProfileRelationship, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.ProfileRelationship{}, err
	}
	return u.relationships.Create(ctx, userID, req)
}

// List возвращает связи текущего пользователя
func (u *UseCase) List(
	ctx context.Context,
	sessionToken string,
	req domain.ListRelationshipsRequest,
) (domain.RelationshipsResponse, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.RelationshipsResponse{}, err
	}
	return u.relationships.List(ctx, userID, req)
}

// Types возвращает виды активных связей текущего пользователя
func (u *UseCase) Types(ctx context.Context, sessionToken string) (domain.RelationshipTypes, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.RelationshipTypes{}, err
	}
	return u.relationships.Types(ctx, userID)
}

// Requests возвращает входящие запросы на связь
func (u *UseCase) Requests(
	ctx context.Context,
	sessionToken string,
	typeName string,
	cursor int64,
	limit int,
) (domain.RelationshipsResponse, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.RelationshipsResponse{}, err
	}
	return u.relationships.Requests(ctx, userID, typeName, cursor, limit)
}

// Suggestions возвращает друзей друзей, отсортированных по числу общих связей
func (u *UseCase) Suggestions(
	ctx context.Context,
	sessionToken string,
	cursor int64,
	limit int,
) (domain.RelationshipSuggestionsResponse, error) {
	userID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.RelationshipSuggestionsResponse{}, err
	}
	return u.relationships.Suggestions(ctx, userID, cursor, limit)
}

// Accept подтверждает входящий запрос на связь
func (u *UseCase) Accept(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.AcceptRelationshipRequest,
) (domain.ProfileRelationship, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.ProfileRelationship{}, err
	}
	return u.relationships.Accept(ctx, userID, req)
}

// Cancel отменяет запрос или закрывает активную связь
func (u *UseCase) Cancel(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CancleRelationshipRequest,
) (domain.ProfileRelationship, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.ProfileRelationship{}, err
	}
	return u.relationships.Cancel(ctx, userID, req)
}
