package post

import (
	"context"
	"errors"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized                = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF                 = sessionusecase.ErrInvalidCSRF
	ErrForbidden                   = errors.New("post belongs to another profile")
	ErrInvalidPost                 = errors.New("post text must be at most 2048 characters")
	ErrPostNotFound                = repository.ErrPostNotFound
	ErrProfileNotFound             = repository.ErrProfileNotFound
	ErrCommunityNotFound           = repository.ErrCommunityNotFound
	ErrCommunityForbidden          = repository.ErrCommunityForbidden
	ErrCommunityServiceUnavailable = repository.ErrCommunityServiceUnavailable
	ErrPostLikeUnavailable         = errors.New("post like repository unavailable")
)

type postRepository interface {
	Create(context.Context, repository.CreatePostParams) (domain.Post, error)
	ReadByID(context.Context, int64) (domain.Post, error)
	UpdateByUUID(context.Context, string, int64, repository.UpdatePostParams) (domain.Post, error)
	DeleteByUUID(context.Context, string, int64) error
	ReadProfilePostsByUUID(context.Context, string, string, int) ([]domain.Post, string, bool, error)
	ReadProfilePosts(context.Context, int64, int64, int) ([]domain.Post, int64, bool, error)
	Update(context.Context, int64, repository.UpdatePostParams) (domain.Post, error)
	Delete(context.Context, int64) error
}

type communityPostRepository interface {
	Create(context.Context, repository.CreatePostParams) (domain.Post, error)
	UpdateCommunityPostByUUID(context.Context, string, int64, repository.UpdatePostParams) (domain.Post, error)
	DeleteCommunityPostByUUID(context.Context, string, int64) error
	ReadCommunityPostsByUID(context.Context, string, string, int) ([]domain.Post, string, bool, error)
}

type communityPostAuthorizer interface {
	CommunityIDByUID(context.Context, string) (int64, error)
	AuthorizePostWrite(context.Context, int64, string) (int64, string, error)
}

type postLikesRepository interface {
	AttachLikeStats(context.Context, []domain.Post, int64) error
	LikeByUUID(context.Context, int64, string) error
	UnlikeByUUID(context.Context, int64, string) error
	ListPostFriendLikers(context.Context, int64, string, string, int) (domain.PostLikersResponse, error)
}

// UseCase содержит бизнес-логику публикаций
type UseCase struct {
	posts          postRepository
	communityPosts communityPostRepository
	communities    communityPostAuthorizer
	likes          postLikesRepository
	sessions       *sessionusecase.UseCase
}

// New создаёт usecase публикаций
func New(posts postRepository, sessions *sessionusecase.UseCase, communities ...communityPostAuthorizer) *UseCase {
	useCase := &UseCase{posts: posts, sessions: sessions}
	if repository, ok := posts.(communityPostRepository); ok {
		useCase.communityPosts = repository
	}
	if repository, ok := posts.(postLikesRepository); ok {
		useCase.likes = repository
	}
	if len(communities) > 0 {
		useCase.communities = communities[0]
	}
	return useCase
}

// Create создаёт публикацию от имени текущего пользователя
func (u *UseCase) Create(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CreatePostRequest,
) (domain.Post, error) {
	userID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Post{}, err
	}
	text := strings.TrimSpace(req.PostText)
	if len([]rune(text)) > 2048 {
		return domain.Post{}, ErrInvalidPost
	}
	var postText *string
	if text != "" {
		postText = &text
	}
	post, err := u.posts.Create(ctx, repository.CreatePostParams{
		PostText:        postText,
		AuthorProfileID: &userID,
	})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "create post", err)
	}
	return post, err
}

// Update изменяет текст собственной публикации
func (u *UseCase) Update(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.UpdatePostRequest,
) (domain.Post, error) {
	userID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Post{}, err
	}
	if req.PostText == nil || len([]rune(strings.TrimSpace(*req.PostText))) > 2048 {
		return domain.Post{}, ErrInvalidPost
	}
	post, err := u.posts.ReadByID(ctx, req.PostID)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "read post for update", err)
		return domain.Post{}, err
	}
	if post.AuthorProfileID == nil || *post.AuthorProfileID != userID {
		return domain.Post{}, ErrForbidden
	}
	text := strings.TrimSpace(*req.PostText)
	updated, err := u.posts.Update(ctx, req.PostID, repository.UpdatePostParams{PostText: &text})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "update post", err)
	}
	return updated, err
}

// UpdateByUID изменяет текст собственной публикации по UUID
func (u *UseCase) UpdateByUID(ctx context.Context, sessionToken, csrfToken string, req domain.UpdatePostRequest) (domain.Post, error) {
	userID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Post{}, err
	}
	if req.PostText == nil || len([]rune(strings.TrimSpace(*req.PostText))) > 2048 {
		return domain.Post{}, ErrInvalidPost
	}
	text := strings.TrimSpace(*req.PostText)
	post, err := u.posts.UpdateByUUID(ctx, req.PostUID, userID, repository.UpdatePostParams{PostText: &text})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "update post", err)
		return domain.Post{}, err
	}
	if err := u.attachLikeStats(ctx, []domain.Post{post}, userID); err != nil {
		return domain.Post{}, err
	}
	return post, nil
}

// Delete удаляет собственную публикацию
func (u *UseCase) Delete(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	postID int64,
) error {
	userID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	post, err := u.posts.ReadByID(ctx, postID)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "read post for delete", err)
		return err
	}
	if post.AuthorProfileID == nil || *post.AuthorProfileID != userID {
		return ErrForbidden
	}
	err = u.posts.Delete(ctx, postID)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "delete post", err)
	}
	return err
}

// DeleteByUID удаляет собственную публикацию по UUID
func (u *UseCase) DeleteByUID(ctx context.Context, sessionToken, csrfToken, postUID string) error {
	userID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	if err := u.posts.DeleteByUUID(ctx, postUID, userID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "delete post", err)
		return err
	}
	return nil
}

// UserPosts возвращает публикации указанного пользователя
func (u *UseCase) UserPosts(
	ctx context.Context,
	profileID int64,
	cursor int64,
	limit int,
) (domain.UserPostsResponse, error) {
	posts, nextCursor, hasMore, err := u.posts.ReadProfilePosts(ctx, profileID, cursor, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "read profile posts", err)
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

// UserPostsByUID возвращает публикации профиля по UUID
func (u *UseCase) UserPostsByUID(ctx context.Context, profileUID, cursorUID string, limit int, sessionToken ...string) (domain.UserPostsResponse, error) {
	posts, nextUID, hasMore, err := u.posts.ReadProfilePostsByUUID(ctx, profileUID, cursorUID, limit)
	if err != nil {
		return domain.UserPostsResponse{}, err
	}
	viewerID := u.optionalViewerID(sessionToken)
	if err := u.attachLikeStats(ctx, posts, viewerID); err != nil {
		return domain.UserPostsResponse{}, err
	}
	return domain.UserPostsResponse{Posts: posts, ProfileUID: profileUID, CursorUID: cursorUID, Limit: limit, HasMore: hasMore, NextCursorUID: nextUID}, nil
}

// CreateCommunityPost публикует запись от имени сообщества после проверки роли участника.
func (u *UseCase) CreateCommunityPost(ctx context.Context, session, csrf string, req domain.CommunityPostRequest) (domain.Post, error) {
	profileID, err := u.authorize(session, csrf)
	if err != nil {
		return domain.Post{}, err
	}
	if u.communities == nil || u.communityPosts == nil {
		return domain.Post{}, ErrCommunityServiceUnavailable
	}
	if req.PostText != nil && len([]rune(strings.TrimSpace(*req.PostText))) > 2048 {
		return domain.Post{}, ErrInvalidPost
	}
	communityID, name, err := u.communities.AuthorizePostWrite(ctx, profileID, req.CommunityUID)
	if err != nil {
		return domain.Post{}, err
	}
	var text *string
	if req.PostText != nil {
		clean := strings.TrimSpace(*req.PostText)
		if clean != "" {
			text = &clean
		}
	}
	post, err := u.communityPosts.Create(ctx, repository.CreatePostParams{PostText: text, AuthorCommunityID: &communityID})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "create community post", err)
		return domain.Post{}, err
	}
	post.Author = domain.Author{Name: name, Type: domain.AuthorCommunity}
	post.AuthorCommunityUID = &req.CommunityUID
	return post, nil
}

// UpdateCommunityPost изменяет публикацию сообщества для владельца, администратора или модератора.
func (u *UseCase) UpdateCommunityPost(ctx context.Context, session, csrf string, req domain.UpdateCommunityPostRequest) (domain.Post, error) {
	profileID, err := u.authorize(session, csrf)
	if err != nil {
		return domain.Post{}, err
	}
	if u.communities == nil || u.communityPosts == nil {
		return domain.Post{}, ErrCommunityServiceUnavailable
	}
	if req.PostText == nil || len([]rune(strings.TrimSpace(*req.PostText))) > 2048 {
		return domain.Post{}, ErrInvalidPost
	}
	communityID, name, err := u.communities.AuthorizePostWrite(ctx, profileID, req.CommunityUID)
	if err != nil {
		return domain.Post{}, err
	}
	text := strings.TrimSpace(*req.PostText)
	post, err := u.communityPosts.UpdateCommunityPostByUUID(ctx, req.PostUID, communityID, repository.UpdatePostParams{PostText: &text})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "update community post", err)
	}
	if err == nil {
		post.Author = domain.Author{Name: name, Type: domain.AuthorCommunity}
		post.AuthorCommunityUID = &req.CommunityUID
		if err = u.attachLikeStats(ctx, []domain.Post{post}, profileID); err != nil {
			return domain.Post{}, err
		}
	}
	return post, err
}

// DeleteCommunityPost удаляет публикацию сообщества для владельца, администратора или модератора.
func (u *UseCase) DeleteCommunityPost(ctx context.Context, session, csrf, communityUID, postUID string) error {
	profileID, err := u.authorize(session, csrf)
	if err != nil {
		return err
	}
	if u.communities == nil || u.communityPosts == nil {
		return ErrCommunityServiceUnavailable
	}
	communityID, _, err := u.communities.AuthorizePostWrite(ctx, profileID, communityUID)
	if err != nil {
		return err
	}
	if err := u.communityPosts.DeleteCommunityPostByUUID(ctx, postUID, communityID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "delete community post", err)
		return err
	}
	return nil
}

// CommunityPosts возвращает страницу публикаций сообщества.
func (u *UseCase) CommunityPosts(ctx context.Context, communityUID, cursorUID string, limit int, sessionToken ...string) (domain.CommunityPostsResponse, error) {
	if u.communityPosts == nil {
		return domain.CommunityPostsResponse{}, ErrCommunityServiceUnavailable
	}
	posts, nextUID, hasMore, err := u.communityPosts.ReadCommunityPostsByUID(ctx, communityUID, cursorUID, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "list community posts", err)
		return domain.CommunityPostsResponse{}, err
	}
	if err := u.attachLikeStats(ctx, posts, u.optionalViewerID(sessionToken)); err != nil {
		return domain.CommunityPostsResponse{}, err
	}
	return domain.CommunityPostsResponse{Posts: posts, CommunityUID: communityUID, Cursor: cursorUID, Limit: limit, HasMore: hasMore, NextCursor: nextUID}, nil
}

// Like ставит лайк публикации от имени текущего пользователя.
func (u *UseCase) Like(ctx context.Context, sessionToken, csrfToken, postUID string) error {
	profileID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	if u.likes == nil {
		return ErrPostLikeUnavailable
	}
	if err := u.likes.LikeByUUID(ctx, profileID, postUID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "like post", err)
		return err
	}
	return nil
}

// Unlike снимает лайк текущего пользователя с публикации.
func (u *UseCase) Unlike(ctx context.Context, sessionToken, csrfToken, postUID string) error {
	profileID, err := u.authorize(sessionToken, csrfToken)
	if err != nil {
		return err
	}
	if u.likes == nil {
		return ErrPostLikeUnavailable
	}
	if err := u.likes.UnlikeByUUID(ctx, profileID, postUID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "unlike post", err)
		return err
	}
	return nil
}

// PostLikers возвращает страницу друзей текущего пользователя, поставивших лайк.
func (u *UseCase) PostLikers(ctx context.Context, sessionToken, postUID, cursorUID string, limit int) (domain.PostLikersResponse, error) {
	if u.sessions == nil {
		return domain.PostLikersResponse{}, ErrUnauthorized
	}
	profileID, _, err := u.sessions.Read(sessionToken)
	if err != nil {
		return domain.PostLikersResponse{}, err
	}
	if u.likes == nil {
		return domain.PostLikersResponse{}, ErrPostLikeUnavailable
	}
	response, err := u.likes.ListPostFriendLikers(ctx, profileID, postUID, cursorUID, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "list post friend likers", err)
	}
	return response, err
}

func (u *UseCase) optionalViewerID(sessionToken []string) int64 {
	if u.sessions == nil || len(sessionToken) == 0 || sessionToken[0] == "" {
		return 0
	}
	profileID, _, err := u.sessions.Read(sessionToken[0])
	if err != nil {
		return 0
	}
	return profileID
}

func (u *UseCase) attachLikeStats(ctx context.Context, posts []domain.Post, viewerID int64) error {
	if u.likes == nil {
		return nil
	}
	if err := u.likes.AttachLikeStats(ctx, posts, viewerID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "post", "attach like stats", err)
		return err
	}
	return nil
}

func (u *UseCase) authorize(sessionToken, csrfToken string) (int64, error) {
	return u.sessions.Authorize(sessionToken, csrfToken)
}
