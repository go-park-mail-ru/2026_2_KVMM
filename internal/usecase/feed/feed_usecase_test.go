package feed

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	feedmock "kvmm/internal/mocks/feed"
)

func TestList(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := feedmock.NewMockpostRepository(ctrl)
	posts.EXPECT().ReadFeedPosts(gomock.Any(), int64(1), 10).Return([]domain.Post{{ID: 1}}, int64(2), true, nil)
	response, err := New(posts).List(context.Background(), 1, 10)
	if err != nil || len(response.Posts) != 1 || response.NextCursor != 2 || !response.HasMore {
		t.Fatalf("List() = %+v, %v", response, err)
	}
}

func TestListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := feedmock.NewMockpostRepository(ctrl)
	posts.EXPECT().ReadFeedPosts(gomock.Any(), int64(0), 10).Return(nil, int64(0), false, errors.New("database unavailable"))
	if _, err := New(posts).List(context.Background(), 0, 10); err == nil {
		t.Fatal("List() expected error")
	}
}
