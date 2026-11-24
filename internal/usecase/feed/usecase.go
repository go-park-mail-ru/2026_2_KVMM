package feed

import (
	"context"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	sessionusecase "kvmm/internal/usecase/session"
)

type likeStatsReader interface {
	AttachLikeStats(context.Context, []domain.Post, int64) error
}

type postRepository interface {
	ReadFeedPosts(context.Context, int64, int) ([]domain.Post, int64, bool, error)
	ReadFeedPostsByUUID(context.Context, string, int) ([]domain.Post, string, bool, error)
}

// UseCase содержит сценарий получения ленты
type UseCase struct {
	posts    postRepository
	sessions *sessionusecase.UseCase
}

// New создаёт usecase ленты
func New(posts postRepository, sessions ...*sessionusecase.UseCase) *UseCase {
	useCase := &UseCase{posts: posts}
	if len(sessions) > 0 {
		useCase.sessions = sessions[0]
	}
	return useCase
}

// List возвращает страницу ленты публикаций
func (u *UseCase) List(ctx context.Context, cursor int64, limit int) (domain.FeedResponse, error) {
	posts, nextCursor, hasMore, err := u.posts.ReadFeedPosts(ctx, cursor, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "feed", "read feed posts", err)
		return domain.FeedResponse{}, err
	}
	if reader, ok := u.posts.(likeStatsReader); ok {
		if err := reader.AttachLikeStats(ctx, posts, 0); err != nil {
			logging.LogUseCaseErrorContext(ctx, "feed", "attach like stats", err)
			return domain.FeedResponse{}, err
		}
	}
	return domain.FeedResponse{
		Posts:      posts,
		Cursor:     cursor,
		Limit:      limit,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

// ListByUID возвращает страницу ленты с UUID-курсором
func (u *UseCase) ListByUID(ctx context.Context, cursorUID string, limit int, sessionToken ...string) (domain.FeedResponse, error) {
	posts, nextUID, hasMore, err := u.posts.ReadFeedPostsByUUID(ctx, cursorUID, limit)
	if err != nil {
		return domain.FeedResponse{}, err
	}
	if reader, ok := u.posts.(likeStatsReader); ok {
		var viewerID int64
		if u.sessions != nil && len(sessionToken) > 0 && sessionToken[0] != "" {
			if id, _, readErr := u.sessions.Read(sessionToken[0]); readErr == nil {
				viewerID = id
			}
		}
		if err := reader.AttachLikeStats(ctx, posts, viewerID); err != nil {
			logging.LogUseCaseErrorContext(ctx, "feed", "attach like stats", err)
			return domain.FeedResponse{}, err
		}
	}
	return domain.FeedResponse{Posts: posts, CursorUID: cursorUID, Limit: limit, HasMore: hasMore, NextCursorUID: nextUID}, nil
}
