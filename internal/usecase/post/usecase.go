package post

import (
	"context"
	"errors"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF  = sessionusecase.ErrInvalidCSRF
	ErrForbidden    = errors.New("post belongs to another profile")
	ErrInvalidPost  = errors.New("post text must be at most 256 characters") // TODO: мб надо увеличить
)

type postRepository interface {
	Create(context.Context, repository.CreatePostParams) (domain.Post, error)
	ReadByID(context.Context, int64) (domain.Post, error)
	ReadProfilePosts(context.Context, int64, int64, int) ([]domain.Post, int64, bool, error)
	Update(context.Context, int64, repository.UpdatePostParams) (domain.Post, error)
	Delete(context.Context, int64) error
}

// UseCase содержит бизнес-логику публикаций
type UseCase struct {
	posts    postRepository
	sessions *sessionusecase.UseCase
}

// New создаёт usecase публикаций
func New(posts postRepository, sessions *sessionusecase.UseCase) *UseCase {
	return &UseCase{posts: posts, sessions: sessions}
}

// Create создаёт публикацию от имени текущего пользователя
func (u *UseCase) Create(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CreatePostRequest,
) (domain.Post, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Post{}, err
	}
	text := strings.TrimSpace(req.PostText)
	if len([]rune(text)) > 256 {
		return domain.Post{}, ErrInvalidPost
	}
	var postText *string
	if text != "" {
		postText = &text
	}
	return u.posts.Create(ctx, repository.CreatePostParams{
		PostText:        postText,
		AuthorProfileID: &userID,
	})
}

// Update изменяет текст собственной публикации
func (u *UseCase) Update(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.UpdatePostRequest,
) (domain.Post, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Post{}, err
	}
	if req.PostText == nil || len([]rune(strings.TrimSpace(*req.PostText))) > 256 {
		return domain.Post{}, ErrInvalidPost
	}
	post, err := u.posts.ReadByID(ctx, req.PostID)
	if err != nil {
		return domain.Post{}, err
	}
	if post.AuthorProfileID == nil || *post.AuthorProfileID != userID {
		return domain.Post{}, ErrForbidden
	}
	text := strings.TrimSpace(*req.PostText)
	return u.posts.Update(ctx, req.PostID, repository.UpdatePostParams{PostText: &text})
}

// Delete удаляет собственную публикацию
func (u *UseCase) Delete(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	postID int64,
) error {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	post, err := u.posts.ReadByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.AuthorProfileID == nil || *post.AuthorProfileID != userID {
		return ErrForbidden
	}
	return u.posts.Delete(ctx, postID)
}

// UserPosts возвращает публикации пользователя
func (u *UseCase) UserPosts(
	ctx context.Context,
	profileID int64,
	cursor int64,
	limit int,
) (domain.UserPostsResponse, error) {
	posts, nextCursor, hasMore, err := u.posts.ReadProfilePosts(ctx, profileID, cursor, limit)
	if err != nil {
		return domain.UserPostsResponse{}, err
	}
	return domain.UserPostsResponse{
		Posts:      posts,
		ProfileID:  profileID,
		Cursor:     cursor,
		Limit:      limit,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}
