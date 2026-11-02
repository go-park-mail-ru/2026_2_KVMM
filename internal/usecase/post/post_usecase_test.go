package post

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	postmock "kvmm/internal/mocks/post"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

func TestPostUseCaseCRUD(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := postmock.NewMockpostRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(posts, sessionusecase.New(session))
	ctx := context.Background()
	authorID := int64(7)
	post := domain.Post{ID: 1, AuthorProfileID: &authorID}

	session.EXPECT().Read("session").Return(authorID, "csrf", nil).Times(3)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	posts.EXPECT().Create(ctx, gomock.Any()).Return(post, nil)
	posts.EXPECT().ReadByID(ctx, int64(1)).Return(post, nil).Times(2)
	posts.EXPECT().Update(ctx, int64(1), gomock.Any()).Return(post, nil)
	posts.EXPECT().Delete(ctx, int64(1)).Return(nil)
	posts.EXPECT().ReadProfilePosts(ctx, int64(7), int64(0), 10).Return([]domain.Post{post}, int64(2), true, nil)

	if _, err := useCase.Create(ctx, "session", "csrf", domain.CreatePostRequest{PostText: "hello"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	text := "updated"
	if _, err := useCase.Update(ctx, "session", "csrf", domain.UpdatePostRequest{PostID: 1, PostText: &text}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := useCase.Delete(ctx, "session", "csrf", 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if response, err := useCase.UserPosts(ctx, 7, 0, 10); err != nil || len(response.Posts) != 1 {
		t.Fatalf("UserPosts() = %+v, %v", response, err)
	}
}

func TestPostUseCaseErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := postmock.NewMockpostRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(posts, sessionusecase.New(session))
	ctx := context.Background()
	authorID := int64(7)
	foreignID := int64(8)
	post := domain.Post{ID: 1, AuthorProfileID: &foreignID}

	longText := make([]rune, 257)
	for i := range longText {
		longText[i] = 'a'
	}
	session.EXPECT().Read("session").Return(authorID, "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	if _, err := useCase.Create(ctx, "session", "csrf", domain.CreatePostRequest{PostText: string(longText)}); !errors.Is(err, ErrInvalidPost) {
		t.Fatalf("Create(long) error = %v", err)
	}
	session.EXPECT().Read("session").Return(authorID, "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	posts.EXPECT().ReadByID(ctx, int64(1)).Return(post, nil)
	text := "x"
	if _, err := useCase.Update(ctx, "session", "csrf", domain.UpdatePostRequest{PostID: 1, PostText: &text}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Update(foreign) error = %v", err)
	}
	posts.EXPECT().ReadProfilePosts(ctx, int64(7), int64(0), 10).Return(nil, int64(0), false, repository.ErrProfileNotFound)
	if _, err := useCase.UserPosts(ctx, 7, 0, 10); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("UserPosts() error = %v", err)
	}
}
