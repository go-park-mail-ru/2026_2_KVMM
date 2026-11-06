package relationship

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	relationshipmock "kvmm/internal/mocks/relationship"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

func TestRelationshipUseCaseMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	relationships := relationshipmock.NewMockrelationshipRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(relationships, sessionusecase.New(session))
	ctx := context.Background()
	relationship := domain.ProfileRelationship{ProfileIDMaster: 7, ProfileIDSlave: 2}

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).Times(7)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	relationships.EXPECT().Create(ctx, int64(7), gomock.Any()).Return(relationship, nil)
	relationships.EXPECT().List(ctx, int64(7), gomock.Any()).Return(domain.RelationshipsResponse{}, nil)
	relationships.EXPECT().Types(ctx, int64(7)).Return(domain.RelationshipTypes{}, nil)
	relationships.EXPECT().Requests(ctx, int64(7), "friendship", int64(0), 10).Return(domain.RelationshipsResponse{}, nil)
	relationships.EXPECT().Suggestions(ctx, int64(7), int64(0), 10).Return(domain.RelationshipSuggestionsResponse{}, nil)
	relationships.EXPECT().Accept(ctx, int64(7), gomock.Any()).Return(relationship, nil)
	relationships.EXPECT().Cancel(ctx, int64(7), gomock.Any()).Return(relationship, nil)

	if _, err := useCase.Create(ctx, "session", "csrf", domain.CreateRelationshipRequest{RelationshipsType: domain.RelationshipTypeFriendship}); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.List(ctx, "session", domain.ListRelationshipsRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Types(ctx, "session"); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Requests(ctx, "session", "friendship", 0, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Suggestions(ctx, "session", 0, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Accept(ctx, "session", "csrf", domain.AcceptRelationshipRequest{RelationshipsType: domain.RelationshipTypeFriendship}); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.Cancel(ctx, "session", "csrf", domain.CancleRelationshipRequest{RelationshipsType: domain.RelationshipTypeFriendship}); err != nil {
		t.Fatal(err)
	}
}

func TestRelationshipUseCaseRejectsUnsupportedType(t *testing.T) {
	ctrl := gomock.NewController(t)
	relationships := relationshipmock.NewMockrelationshipRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(relationships, sessionusecase.New(session))

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	if _, err := useCase.Create(context.Background(), "session", "csrf", domain.CreateRelationshipRequest{
		RelationshipsType: "blocked",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Create(unsupported type) error = %v", err)
	}
}

func TestRelationshipUseCaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	relationships := relationshipmock.NewMockrelationshipRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(relationships, sessionusecase.New(session))
	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	relationships.EXPECT().Types(gomock.Any(), int64(7)).Return(domain.RelationshipTypes{}, repository.ErrRelationshipNotFound)
	if _, err := useCase.Types(context.Background(), "session"); !errors.Is(err, repository.ErrRelationshipNotFound) {
		t.Fatalf("Types() error = %v", err)
	}
}
