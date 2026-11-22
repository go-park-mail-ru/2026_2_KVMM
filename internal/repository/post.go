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
	ErrPostNotFound              = errors.New("post not found")
	ErrInvalidPostAuthor         = errors.New("exactly one post author is required")
	ErrProfileServiceUnavailable = errors.New("profile service is unavailable")
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
		AND post.author_profile_id IS NOT NULL
	ORDER BY post.id DESC
	LIMIT $2`

// PostRepository выполняет SQL-операции с публикациями и запрашивает профили через User service.
type profileServiceReader interface {
	GetProfileIDByUID(context.Context, string) (int64, error)
	GetProfilesByIDs(context.Context, []int64) ([]domain.Profile, error)
}

type PostRepository struct {
	pool     dbPool
	profiles profileServiceReader
}

// NewPostRepository создаёт репозиторий публикаций и клиента сервиса профилей.
func NewPostRepository(pool dbPool, profiles ...profileServiceReader) *PostRepository {
	repository := &PostRepository{pool: pool}
	if len(profiles) > 0 {
		repository.profiles = profiles[0]
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
	posts := []domain.Post{post}
	if err := r.attachPostAuthors(ctx, posts); err != nil {
		return domain.Post{}, err
	}
	return posts[0], nil
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

// ReadFeedPostsByUUID возвращает ленту с UUID-курсором без предварительного поиска ID.
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
			AND post.author_profile_id IS NOT NULL


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
			AND post.author_profile_id IS NOT NULL
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
	for _, post := range posts {
		if post.AuthorProfileID == nil {
			continue
		}
		if _, ok := seen[*post.AuthorProfileID]; ok {
			continue
		}
		seen[*post.AuthorProfileID] = struct{}{}
		profileIDs = append(profileIDs, *post.AuthorProfileID)
	}
	if len(profileIDs) == 0 {
		return nil
	}
	if r.profiles == nil {
		return ErrProfileServiceUnavailable
	}
	profiles, err := r.profiles.GetProfilesByIDs(ctx, profileIDs)
	if err != nil {
		return err
	}
	authors := make(map[int64]domain.Author, len(profiles))
	profileUIDs := make(map[int64]string, len(profiles))
	for _, profile := range profiles {
		name := strings.TrimSpace(strings.TrimSpace(derefString(profile.Surname)) + " " + strings.TrimSpace(derefString(profile.ProfileName)))
		if name == "" {
			name = profile.Nickname
		}
		var avatarURL string
		if len(profile.AvatarURLs) > 0 {
			avatarURL = profile.AvatarURLs[0]
		}
		authors[profile.ID] = domain.Author{Name: name, AvatarURL: avatarURL, Type: domain.AuthorProfile}
		profileUIDs[profile.ID] = profile.UID
	}
	for index := range posts {
		if posts[index].AuthorProfileID == nil {
			continue
		}
		if author, ok := authors[*posts[index].AuthorProfileID]; ok {
			posts[index].Author = author
			if uid := profileUIDs[*posts[index].AuthorProfileID]; uid != "" {
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
