package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrPostNotFound                = errors.New("post not found")
	ErrInvalidPostAuthor           = errors.New("exactly one post author is required")
	ErrProfileServiceUnavailable   = errors.New("profile service is unavailable")
	ErrCommunityServiceUnavailable = errors.New("community service is unavailable")
)

const postColumns = `
	post.id,
	post.uid,
	post.post_text,
	post.author_profile_id,
	post.author_community_id,
	post.created_at,
	post.updated_at,
	post.deleted_at`

const postReturningColumns = `
	id,
	uid,
	post_text,
	author_profile_id,
	author_community_id,
	created_at,
	updated_at,
	deleted_at`

const updatePostByUUIDQuery = `
	UPDATE post
	SET post_text = $1, updated_at = NOW()
	WHERE uid = $2::uuid
		AND author_profile_id = $3
		AND deleted_at IS NULL
	RETURNING ` + postReturningColumns

const deletePostByUUIDQuery = `
	UPDATE post
	SET deleted_at = NOW()
	WHERE uid = $1::uuid
		AND author_profile_id = $2
		AND deleted_at IS NULL`

const updateCommunityPostByUUIDQuery = `
	UPDATE post SET post_text = $1, updated_at = NOW()
	WHERE uid = $2::uuid AND author_community_id = $3 AND deleted_at IS NULL
	RETURNING ` + postReturningColumns

const deleteCommunityPostByUUIDQuery = `
	UPDATE post SET deleted_at = NOW(), updated_at = NOW()
	WHERE uid = $1::uuid AND author_community_id = $2 AND deleted_at IS NULL`

const readCommunityPostsQuery = `
	SELECT ` + postColumns + ` FROM post
	WHERE author_community_id = $1 AND deleted_at IS NULL
		AND ($2 = '' OR post.id < (SELECT id FROM post WHERE uid = NULLIF($2, '')::uuid))
	ORDER BY post.id DESC LIMIT $3`

const readProfilePostsByUUIDQuery = `
	SELECT ` + postColumns + `
	FROM post
	WHERE post.author_profile_id = $1
		AND post.deleted_at IS NULL
		AND ($2 = '' OR post.id < (
			SELECT id FROM post WHERE uid = NULLIF($2, '')::uuid
		))
	ORDER BY post.id DESC
	LIMIT $3`

const readFeedPostsByUUIDQuery = `
	SELECT ` + postColumns + `
	FROM post
	WHERE post.deleted_at IS NULL
		AND ($1 = '' OR post.id < (
			SELECT id FROM post WHERE uid = NULLIF($1, '')::uuid
		))
	ORDER BY post.id DESC
	LIMIT $2`

const postLikeSummaryQuery = `
		SELECT post_id, COUNT(*)::BIGINT, COALESCE(BOOL_OR(profile_id = $2), FALSE)
		FROM profile_like
		WHERE post_id = ANY($1::BIGINT[]) AND deleted_at IS NULL
		GROUP BY post_id`
const likePostByUUIDQuery = `
		INSERT INTO profile_like (profile_id, post_id)
		SELECT $1, post.id
		FROM post
		WHERE post.uid = $2::uuid AND post.deleted_at IS NULL
		ON CONFLICT (profile_id, post_id) WHERE deleted_at IS NULL
		DO UPDATE SET deleted_at = NULL
		RETURNING id`
const readPostIDByUUIDQuery = `
		SELECT id FROM post WHERE uid = $1::uuid AND deleted_at IS NULL`
const removePostLikeByUUIDQuery = `
		UPDATE profile_like
		SET deleted_at = NOW()
		WHERE profile_id = $1
			AND post_id = (SELECT id FROM post WHERE uid = $2::uuid AND deleted_at IS NULL)
			AND deleted_at IS NULL`
const listPostFriendLikersQuery = `
		SELECT liked.profile_id
		FROM post target
		JOIN profile_like liked ON liked.post_id = target.id AND liked.deleted_at IS NULL
		WHERE target.uid = $1::uuid AND target.deleted_at IS NULL
			AND liked.profile_id <> $2
			AND liked.profile_id > $3
			AND EXISTS (
				SELECT 1
				FROM profile_relationship relationship
				WHERE relationship.relationships_status = 'accepted'
					AND relationship.relationships_type = 'friendship'
					AND relationship.deleted_at IS NULL
					AND (
						(relationship.profile_id_master = $2 AND relationship.profile_id_slave = liked.profile_id)
						OR (relationship.profile_id_slave = $2 AND relationship.profile_id_master = liked.profile_id)
					)
			)
		ORDER BY liked.profile_id
		LIMIT $4`

// PostRepository выполняет SQL-операции с публикациями и получает авторов от профильных сервисов
type profileServiceReader interface {
	GetProfileIDByUID(context.Context, string) (int64, error)
	GetProfilesByIDs(context.Context, []int64) ([]domain.Profile, error)
}

type communityServiceReader interface {
	CommunityIDByUID(context.Context, string) (int64, error)
	GetCommunitiesByIDs(context.Context, []int64) ([]domain.Community, error)
}

type PostRepository struct {
	pool        dbPool
	profiles    profileServiceReader
	communities communityServiceReader
}

// NewPostRepository создаёт репозиторий публикаций и клиента сервиса профилей
func NewPostRepository(pool dbPool, profiles profileServiceReader, communities ...communityServiceReader) *PostRepository {
	repository := &PostRepository{pool: pool, profiles: profiles}
	if len(communities) > 0 {
		repository.communities = communities[0]
	}
	return repository
}

// CreatePostParams содержит данные для создания поста
type CreatePostParams struct {
	PostText          *string
	AuthorProfileID   *int64
	AuthorCommunityID *int64
}

// UpdatePostParams содержит поля поста, которые нужно изменить
type UpdatePostParams struct {
	PostText *string
}

// Create создаёт пост и возвращает созданную запись
func (r *PostRepository) Create(
	ctx context.Context,
	params CreatePostParams,
) (domain.Post, error) {
	if (params.AuthorProfileID == nil) == (params.AuthorCommunityID == nil) {
		return domain.Post{}, ErrInvalidPostAuthor
	}

	const createPostQuery = `
		INSERT INTO post (
			post_text,
			author_profile_id,
			author_community_id
		)
		VALUES ($1, $2, $3)
		RETURNING ` + postReturningColumns

	row := r.pool.QueryRow(
		ctx,
		createPostQuery,
		nullableString(params.PostText),
		nullableInt64(params.AuthorProfileID),
		nullableInt64(params.AuthorCommunityID),
	)

	post, err := scanPost(row)
	if err != nil {
		return domain.Post{}, normalizePostError(err)
	}
	if params.AuthorCommunityID != nil {
		return post, nil
	}
	posts := []domain.Post{post}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return domain.Post{}, err
	}

	return posts[0], nil
}

// ReadByID возвращает активный пост по ID
func (r *PostRepository) ReadByID(
	ctx context.Context,
	id int64) (domain.Post, error) {

	const readPostByIDQuery = `
		SELECT ` + postColumns + `
		FROM post
		WHERE post.id = $1
			AND post.deleted_at IS NULL`

	post, err := scanPost(r.pool.QueryRow(ctx, readPostByIDQuery, id))
	if err != nil {
		return domain.Post{}, normalizePostError(err)
	}
	return post, nil
}

// UpdateByUUID изменяет пост текущего пользователя напрямую по UUID
func (r *PostRepository) UpdateByUUID(ctx context.Context, uid string, authorID int64, params UpdatePostParams) (domain.Post, error) {
	post, err := scanPost(r.pool.QueryRow(ctx, updatePostByUUIDQuery, nullableString(params.PostText), uid, authorID))
	if err != nil {
		return domain.Post{}, normalizePostError(err)
	}
	posts := []domain.Post{post}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return domain.Post{}, err
	}
	return posts[0], nil
}

// DeleteByUUID помечает пост текущего пользователя удалённым напрямую по UUID
func (r *PostRepository) DeleteByUUID(ctx context.Context, uid string, authorID int64) error {
	command, err := r.pool.Exec(ctx, deletePostByUUIDQuery, uid, authorID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

// UpdateCommunityPostByUUID изменяет пост сообщества по UUID
func (r *PostRepository) UpdateCommunityPostByUUID(ctx context.Context, uid string, communityID int64, params UpdatePostParams) (domain.Post, error) {
	post, err := scanPost(r.pool.QueryRow(ctx, updateCommunityPostByUUIDQuery, nullableString(params.PostText), uid, communityID))
	if err != nil {
		return domain.Post{}, normalizePostError(err)
	}
	return post, nil
}

// DeleteCommunityPostByUUID мягко удаляет пост сообщества
func (r *PostRepository) DeleteCommunityPostByUUID(ctx context.Context, uid string, communityID int64) error {
	result, err := r.pool.Exec(ctx, deleteCommunityPostByUUIDQuery, uid, communityID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

// ReadCommunityPostsByUID возвращает страницу публикаций по UUID сообщества
func (r *PostRepository) ReadCommunityPostsByUID(ctx context.Context, communityUID, cursorUID string, limit int) ([]domain.Post, string, bool, error) {
	if r.communities == nil {
		return nil, "", false, ErrCommunityServiceUnavailable
	}
	communityID, err := r.communities.CommunityIDByUID(ctx, communityUID)
	if err != nil {
		return nil, "", false, err
	}
	limit = normalizeLimit(limit, 10)
	rows, err := r.pool.Query(ctx, readCommunityPostsQuery, communityID, cursorUID, limit+1)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()
	items := make([]domain.Post, 0, limit+1)
	for rows.Next() {
		item, scanErr := scanPost(rows)
		if scanErr != nil {
			return nil, "", false, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	var next string
	if len(items) > 0 {
		next = items[len(items)-1].UID
	}
	if err := r.attachPostAuthors(ctx, items); err != nil {
		return nil, "", false, err
	}
	return items, next, more, nil
}

// AttachLikeStats загружает счётчики и состояние лайка текущего пользователя для страницы
func (r *PostRepository) AttachLikeStats(ctx context.Context, posts []domain.Post, viewerID int64) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(posts))
	indexes := make(map[int64]int, len(posts))
	for i := range posts {
		posts[i].LikesCount = 0
		posts[i].IsLikedByMe = false
		ids = append(ids, posts[i].ID)
		indexes[posts[i].ID] = i
	}
	rows, err := r.pool.Query(ctx, postLikeSummaryQuery, ids, viewerID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var postID, count int64
		var likedByMe bool
		if err := rows.Scan(&postID, &count, &likedByMe); err != nil {
			return err
		}
		if index, ok := indexes[postID]; ok {
			posts[index].LikesCount = count
			posts[index].IsLikedByMe = likedByMe
		}
	}
	return rows.Err()
}

// LikeByUUID ставит лайк посту  (повторный вызов не создаёт второй активный лайк, а убирает delete at)
func (r *PostRepository) LikeByUUID(ctx context.Context, profileID int64, postUID string) error {
	var likeID int64
	if err := r.pool.QueryRow(ctx, likePostByUUIDQuery, profileID, postUID).Scan(&likeID); err != nil {
		return normalizePostError(err)
	}
	return nil
}

// UnlikeByUUID снимает лайк текущего пользователя с поста
func (r *PostRepository) UnlikeByUUID(ctx context.Context, profileID int64, postUID string) error {
	result, err := r.pool.Exec(ctx, removePostLikeByUUIDQuery, profileID, postUID)
	if err != nil {
		return err
	}
	if result.RowsAffected() > 0 {
		return nil
	}
	var postID int64
	if err := r.pool.QueryRow(ctx, readPostIDByUUIDQuery, postUID).Scan(&postID); err != nil {
		return normalizePostError(err)
	}
	return nil
}

// ListPostFriendLikers возвращает друзей, лайкнувших пост
func (r *PostRepository) ListPostFriendLikers(ctx context.Context, viewerID int64, postUID, cursorUID string, limit int) (domain.PostLikersResponse, error) {
	if r.profiles == nil {
		return domain.PostLikersResponse{}, ErrProfileServiceUnavailable
	}
	limit = normalizeLimit(limit, 50)
	var cursorID int64
	if cursorUID != "" {
		var err error
		cursorID, err = r.profiles.GetProfileIDByUID(ctx, cursorUID)
		if err != nil {
			return domain.PostLikersResponse{}, err
		}
	}
	rows, err := r.pool.Query(ctx, listPostFriendLikersQuery, postUID, viewerID, cursorID, limit+1)
	if err != nil {
		return domain.PostLikersResponse{}, err
	}
	profileIDs := make([]int64, 0, limit+1)
	for rows.Next() {
		var profileID int64
		if err := rows.Scan(&profileID); err != nil {
			rows.Close()
			return domain.PostLikersResponse{}, err
		}
		profileIDs = append(profileIDs, profileID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return domain.PostLikersResponse{}, err
	}
	rows.Close()

	if len(profileIDs) == 0 {
		var postID int64
		if err := r.pool.QueryRow(ctx, readPostIDByUUIDQuery, postUID).Scan(&postID); err != nil {
			return domain.PostLikersResponse{}, normalizePostError(err)
		}
	}
	hasMore := len(profileIDs) > limit
	if hasMore {
		profileIDs = profileIDs[:limit]
	}
	profiles, err := r.profiles.GetProfilesByIDs(ctx, profileIDs)
	if err != nil {
		return domain.PostLikersResponse{}, err
	}
	byID := make(map[int64]domain.Profile, len(profiles))
	for _, profile := range profiles {
		byID[profile.ID] = profile
	}
	shortProfiles := make([]domain.ShortProfile, 0, len(profileIDs))
	for _, profileID := range profileIDs {
		profile, ok := byID[profileID]
		if !ok {
			continue
		}
		var avatarURL *string
		if len(profile.AvatarURLs) > 0 && profile.AvatarURLs[0] != "" {
			avatarURL = &profile.AvatarURLs[0]
		}
		shortProfiles = append(shortProfiles, domain.ShortProfile{
			ID: profile.ID, UID: profile.UID, Nickname: profile.Nickname,
			ProfileName: profile.ProfileName, Surname: profile.Surname,
			Patronymic: profile.Patronymic, AvatarURL: avatarURL,
		})
	}
	var nextCursor string
	if len(shortProfiles) > 0 {
		nextCursor = shortProfiles[len(shortProfiles)-1].UID
	}
	return domain.PostLikersResponse{
		PostUID: postUID, Profiles: shortProfiles, Cursor: cursorUID,
		Limit: limit, HasMore: hasMore, NextCursor: nextCursor,
	}, nil
}

// ReadProfilePostsByUUID получает внутренний ID профиля через User service и возвращает страницу постов по UUID
func (r *PostRepository) ReadProfilePostsByUUID(ctx context.Context, profileUID, cursorUID string, limit int) ([]domain.Post, string, bool, error) {
	if r.profiles == nil {
		return nil, "", false, ErrProfileServiceUnavailable
	}
	profileID, err := r.profiles.GetProfileIDByUID(ctx, profileUID)
	if err != nil {
		return nil, "", false, err
	}
	limit = normalizeLimit(limit, 10)
	rows, err := r.pool.Query(ctx, readProfilePostsByUUIDQuery, profileID, cursorUID, limit+1)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()
	posts := make([]domain.Post, 0, limit)
	for rows.Next() {
		post, scanErr := scanPost(rows)
		if scanErr != nil {
			return nil, "", false, scanErr
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	var nextUID string
	if len(posts) > 0 {
		nextUID = posts[len(posts)-1].UID
	}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return nil, "", false, err
	}
	return posts, nextUID, hasMore, nil
}

// ReadFeedPostsByUUID возвращает ленту
func (r *PostRepository) ReadFeedPostsByUUID(ctx context.Context, cursorUID string, limit int) ([]domain.Post, string, bool, error) {
	limit = normalizeLimit(limit, 100)
	rows, err := r.pool.Query(ctx, readFeedPostsByUUIDQuery, cursorUID, limit+1)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()
	posts := make([]domain.Post, 0, limit)
	for rows.Next() {
		post, scanErr := scanPost(rows)
		if scanErr != nil {
			return nil, "", false, scanErr
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, "", false, err
	}
	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}
	var nextUID string
	if len(posts) > 0 {
		nextUID = posts[len(posts)-1].UID
	}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return nil, "", false, err
	}
	return posts, nextUID, hasMore, nil
}

// ReadProfilePosts возвращает посты профиля с курсорной пагинацией
// limit - количесво постов
// cursor — ID последнего отданного поста
// cursor = 0 — первая страница cursor станет самым новым постом
func (r *PostRepository) ReadProfilePosts(
	ctx context.Context,
	id int64,
	cursor int64,
	limit int,
) ([]domain.Post, int64, bool, error) {
	if id < 1 {
		return nil, 0, false, ErrProfileNotFound
	}

	const readPostQuery = `
		SELECT ` + postColumns + `
		FROM post
		WHERE post.author_profile_id = $1
			AND post.deleted_at IS NULL
			AND ($2 = 0 OR post.id < $2)
		ORDER BY post.id DESC
		LIMIT $3`

	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 10)

	queryLimit := limit + 1

	rows, err := r.pool.Query(ctx, readPostQuery, id, cursor, queryLimit)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	result := make([]domain.Post, 0, limit)

	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, 0, false, err
		}

		result = append(result, post)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	result, nextCursor, hasMore := paginatePosts(result, limit)
	if err := r.attachPostAuthors(ctx, result); err != nil {
		return nil, 0, false, err
	}

	return result, nextCursor, hasMore, nil
}

// ReadRelationshipPosts возвращает посты пользователей,
// с которыми у профиля есть подтверждённая связь
// relationshipTypes — типы связей для фильтрации
// Пустой список означает любой тип связи
// limit - количесво постов
// cursor — ID последнего отданного поста
// cursor = 0 — первая страница cursor станет самым новым постом
func (r *PostRepository) ReadRelationshipPosts(
	ctx context.Context,
	profileID int64,
	cursor int64,
	limit int,
	relationshipTypes []string,
) ([]domain.Post, int64, bool, error) {
	if profileID < 1 {
		return nil, 0, false, ErrProfileNotFound
	}

	const readRelationshipPostsQuery = `
		SELECT ` + postColumns + `
		FROM post
		WHERE post.deleted_at IS NULL
			AND ($2 = 0 OR post.id < $2)
			AND EXISTS (
				SELECT 1
				FROM profile_relationship pr
				WHERE pr.deleted_at IS NULL
					AND pr.relationships_status = 'accepted'
					AND (
						(pr.profile_id_master = $1
							AND pr.profile_id_slave = post.author_profile_id)
						OR
						(pr.profile_id_slave = $1
							AND pr.profile_id_master = post.author_profile_id)
					)
					AND (
						cardinality($4::text[]) = 0
						OR pr.relationships_type = ANY($4::text[])
					)
			)
		ORDER BY post.id DESC
		LIMIT $3`

	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 100)

	if relationshipTypes == nil {
		relationshipTypes = []string{}
	}

	queryLimit := limit + 1

	rows, err := r.pool.Query(
		ctx,
		readRelationshipPostsQuery,
		profileID,
		cursor,
		queryLimit,
		relationshipTypes,
	)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	result := make([]domain.Post, 0, queryLimit)

	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, 0, false, err
		}

		result = append(result, post)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	result, nextCursor, hasMore := paginatePosts(result, limit)
	if err := r.attachPostAuthors(ctx, result); err != nil {
		return nil, 0, false, err
	}

	return result, nextCursor, hasMore, nil
}

// ReadFeedPosts возвращает посты в хронологическом порядке
// limit - количесво постов
// cursor — ID последнего отданного поста
// cursor = 0 — первая страница cursor станет самым новым постом
func (r *PostRepository) ReadFeedPosts(
	ctx context.Context,
	cursor int64,
	limit int,
) ([]domain.Post, int64, bool, error) {
	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 100)

	const readFeedPostsQuery = `
		SELECT ` + postColumns + `
		FROM post
		WHERE post.deleted_at IS NULL
			AND ($1 = 0 OR post.id < $1)
		ORDER BY post.id DESC
		LIMIT $2`

	queryLimit := limit + 1

	rows, err := r.pool.Query(
		ctx,
		readFeedPostsQuery,
		cursor,
		queryLimit,
	)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	result := make([]domain.Post, 0, queryLimit)

	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, 0, false, err
		}

		result = append(result, post)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	result, nextCursor, hasMore := paginatePosts(result, limit)
	if err := r.attachPostAuthors(ctx, result); err != nil {
		return nil, 0, false, err
	}

	return result, nextCursor, hasMore, nil
}

// Update изменяет переданные поля активного поста
func (r *PostRepository) Update(
	ctx context.Context,
	id int64,
	params UpdatePostParams,
) (domain.Post, error) {

	sets := make([]string, 0, 1)
	args := make([]any, 0, 2)

	const updatePostQuery = `
		UPDATE post SET %s
		WHERE id = $%d
			AND deleted_at IS NULL
		RETURNING ` + postReturningColumns

	add := func(column string, value any) {
		args = append(args, value)
		sets = append(
			sets,
			fmt.Sprintf("%s = $%d", column, len(args)),
		)
	}

	if params.PostText != nil {
		add("post_text", nullableString(params.PostText))
	}

	if len(sets) == 0 {
		return r.ReadByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")

	args = append(args, id)

	query := fmt.Sprintf(
		updatePostQuery,
		strings.Join(sets, ", "),
		len(args),
	)

	post, err := scanPost(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return domain.Post{}, normalizePostError(err)
	}
	posts := []domain.Post{post}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return domain.Post{}, err
	}
	return posts[0], nil
}

// Delete помечает пост удалённым (soft delete)
func (r *PostRepository) Delete(
	ctx context.Context,
	id int64,
) error {

	const postDeleteQuery = `
		UPDATE post
		SET deleted_at = NOW()
		WHERE id = $1
		AND deleted_at IS NULL`

	command, err := r.pool.Exec(
		ctx,
		postDeleteQuery,
		id,
	)
	if err != nil {
		return err
	}

	if command.RowsAffected() == 0 {
		return ErrPostNotFound
	}

	return nil
}

// scanPost сериализует ответ базы в domain.Post
func scanPost(row rowScanner) (domain.Post, error) {
	var (
		id                  int64
		uid                 pgtype.UUID
		post_text           pgtype.Text
		author_profile_id   pgtype.Int8
		author_community_id pgtype.Int8
		createdAt           pgtype.Timestamptz
		updatedAt           pgtype.Timestamptz
		deletedAt           pgtype.Timestamptz
	)

	dest := []any{
		&id,
		&uid,
		&post_text,
		&author_profile_id,
		&author_community_id,
		&createdAt,
		&updatedAt,
		&deletedAt,
	}

	if err := row.Scan(dest...); err != nil {
		return domain.Post{}, err
	}

	post := domain.Post{
		ID:                id,
		UID:               uuidString(uid),
		PostText:          textPointer(post_text),
		AuthorProfileID:   int64Pointer(author_profile_id),
		AuthorCommunityID: int64Pointer(author_community_id),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         timestampPointer(updatedAt),
		DeletedAt:         timestampPointer(deletedAt),
	}

	return post, nil
}

// attachPostAuthors получает авторов пакетно только через User service.
func (r *PostRepository) attachPostAuthors(ctx context.Context, posts []domain.Post) error {
	profileIDs := make([]int64, 0, len(posts))
	seen := make(map[int64]struct{}, len(posts))
	communityIDs := make([]int64, 0, len(posts))
	communitySeen := make(map[int64]struct{}, len(posts))
	for _, post := range posts {
		if post.AuthorProfileID != nil {
			if _, ok := seen[*post.AuthorProfileID]; !ok {
				seen[*post.AuthorProfileID] = struct{}{}
				profileIDs = append(profileIDs, *post.AuthorProfileID)
			}
		}
		if post.AuthorCommunityID != nil {
			if _, ok := communitySeen[*post.AuthorCommunityID]; !ok {
				communitySeen[*post.AuthorCommunityID] = struct{}{}
				communityIDs = append(communityIDs, *post.AuthorCommunityID)
			}
		}
	}
	profileAuthors := make(map[int64]domain.Author, len(profileIDs))
	profileUIDs := make(map[int64]string, len(profileIDs))
	if len(profileIDs) > 0 {
		if r.profiles == nil {
			return ErrProfileServiceUnavailable
		}
		profiles, err := r.profiles.GetProfilesByIDs(ctx, profileIDs)
		if err != nil {
			return err
		}
		for _, profile := range profiles {
			name := strings.TrimSpace(strings.TrimSpace(derefString(profile.Surname)) + " " + strings.TrimSpace(derefString(profile.ProfileName)))
			if name == "" {
				name = profile.Nickname
			}
			var avatarURL string
			if len(profile.AvatarURLs) > 0 {
				avatarURL = profile.AvatarURLs[0]
			}
			profileAuthors[profile.ID] = domain.Author{Name: name, AvatarURL: avatarURL, Type: domain.AuthorProfile}
			profileUIDs[profile.ID] = profile.UID
		}
	}
	communityAuthors := make(map[int64]domain.Author, len(communityIDs))
	communityUIDs := make(map[int64]string, len(communityIDs))
	if len(communityIDs) > 0 {
		if r.communities == nil {
			return ErrCommunityServiceUnavailable
		}
		communities, err := r.communities.GetCommunitiesByIDs(ctx, communityIDs)
		if err != nil {
			return err
		}
		for _, community := range communities {
			communityAuthors[community.ID] = domain.Author{Name: community.Name, Type: domain.AuthorCommunity}
			communityUIDs[community.ID] = community.UID
		}
	}
	for index := range posts {
		if posts[index].AuthorCommunityID != nil {
			id := *posts[index].AuthorCommunityID
			if author, ok := communityAuthors[id]; ok {
				posts[index].Author = author
				if uid := communityUIDs[id]; uid != "" {
					posts[index].AuthorCommunityUID = &uid
				}
			}
			continue
		}
		if posts[index].AuthorProfileID == nil {
			continue
		}
		id := *posts[index].AuthorProfileID
		if author, ok := profileAuthors[id]; ok {
			posts[index].Author = author
			if uid := profileUIDs[id]; uid != "" {
				posts[index].AuthorProfileUID = &uid
			}
		}
	}
	return nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizePostError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPostNotFound
	}
	return err
}

// paginatePosts обрабатывает результат курсорной пагинации
func paginatePosts(
	posts []domain.Post,
	limit int,
) ([]domain.Post, int64, bool) {
	hasMore := len(posts) > limit

	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor int64
	if len(posts) > 0 {
		nextCursor = posts[len(posts)-1].ID
	}

	return posts, nextCursor, hasMore
}
