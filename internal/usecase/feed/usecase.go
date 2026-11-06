package feed

import (
	"context"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
)

type postRepository interface {
	ReadFeedPosts(context.Context, int64, int) ([]domain.Post, int64, bool, error)
}

// UseCase содержит сценарий получения ленты.
type UseCase struct {
	posts postRepository
}

// New создаёт usecase ленты.
func New(posts postRepository) *UseCase {
	return &UseCase{posts: posts}
}

// List возвращает страницу ленты публикаций.
func (u *UseCase) List(ctx context.Context, cursor int64, limit int) (domain.FeedResponse, error) {
	posts, nextCursor, hasMore, err := u.posts.ReadFeedPosts(ctx, cursor, limit)
	if err != nil {
		logging.LogUseCaseError("feed", "read feed posts", err)
		return domain.FeedResponse{}, err
	}
	return domain.FeedResponse{
		Posts:      posts,
		Cursor:     int(cursor),
		Limit:      limit,
		HasMore:    hasMore,
		NextCursor: int(nextCursor),
	}, nil
}
