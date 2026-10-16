package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPostNotFound      = errors.New("post not found")
	ErrInvalidPostAuthor = errors.New("exactly one post author is required")
)

const postColumns = `
	post.id,
	post.post_text,
	post.author_profile_id,
	post.author_community_id,
	post.created_at,
	post.updated_at,
	post.deleted_at`

const postAuthorColumns = `
	author.author_name,
	author.avatar_url,
	author.author_type`

const postAuthorJoin = `
	LEFT JOIN LATERAL (
		SELECT
			COALESCE(
				NULLIF(
					CONCAT_WS(
						' ',
						NULLIF(BTRIM(p.surname), ''),
						NULLIF(BTRIM(p.profile_name), '')
					),
					''
				),
				p.nickname
			) AS author_name,
			avatar.media_path AS avatar_url,
			'profile'::TEXT AS author_type
		FROM profile p
		LEFT JOIN LATERAL (
			SELECT m.media_path
			FROM profile_avatar pa
			JOIN media m ON m.id = pa.media_id
			WHERE pa.profile_id = p.id
				AND pa.deleted_at IS NULL
				AND m.deleted_at IS NULL
			ORDER BY pa.created_at DESC, pa.media_id DESC
			LIMIT 1
		) avatar ON TRUE
		WHERE p.id = post.author_profile_id
			AND p.deleted_at IS NULL

		/*
		UNION ALL

		SELECT
			c.community_name AS author_name,
			avatar.media_path AS avatar_url,
			'community'::TEXT AS author_type
		FROM community c
		LEFT JOIN LATERAL (
			SELECT m.media_path
			FROM community_avatar ca
			JOIN media m ON m.id = ca.media_id
			WHERE ca.community_id = c.id
				AND ca.deleted_at IS NULL
				AND m.deleted_at IS NULL
			ORDER BY ca.created_at DESC, ca.media_id DESC
			LIMIT 1
		) avatar ON TRUE
		WHERE c.id = post.author_community_id
			AND c.deleted_at IS NULL
		*/
	) author ON TRUE`

const postReturningColumns = `
	id,
	post_text,
	author_profile_id,
	author_community_id,
	created_at,
	updated_at,
	deleted_at`

// PostRepository выполняет sql-операции с таблицей profile
type PostRepository struct {
	pool *pgxpool.Pool
}

// NewProfileRepository создаёт репозиторий пользователей
func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{pool: pool}
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
		WITH created_post AS (
			INSERT INTO post (
				post_text,
				author_profile_id,
				author_community_id
			)
			VALUES ($1, $2, $3)
			RETURNING ` + postReturningColumns + `
		)
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM created_post AS post
		` + postAuthorJoin

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

	return post, nil
}

// ReadByID возвращает активный пост по ID
func (r *PostRepository) ReadByID(
	ctx context.Context,
	id int64) (domain.Post, error) {

	const readPostByIDQuery = `
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM post
		` + postAuthorJoin + `
		WHERE post.id = $1
			AND post.deleted_at IS NULL`

	post, err := scanPost(r.pool.QueryRow(ctx, readPostByIDQuery, id))
	return post, normalizePostError(err)
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
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM post
		` + postAuthorJoin + `
		WHERE post.author_profile_id = $1
			AND post.deleted_at IS NULL
			AND ($2 = 0 OR post.id < $2)
			AND EXISTS (
				SELECT 1
				FROM profile p
				WHERE p.id = post.author_profile_id
					AND p.deleted_at IS NULL
			)
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
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM post
		` + postAuthorJoin + `
		WHERE post.deleted_at IS NULL
			AND ($2 = 0 OR post.id < $2)
			AND post.author_profile_id IS NOT NULL

			AND EXISTS (
				SELECT 1
				FROM profile p
				WHERE p.id = post.author_profile_id
					AND p.deleted_at IS NULL
			)

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

	return result, nextCursor, hasMore, nil
}

// ReadFeedPosts возвращает посты в хронологическом порядке
// Пустой список означает любой тип связи
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
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM post
		` + postAuthorJoin + `
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
		WITH updated_post AS (
			UPDATE post SET %s
			WHERE id = $%d
				AND deleted_at IS NULL
			RETURNING ` + postReturningColumns + `
		)
		SELECT
			` + postColumns + `,
			` + postAuthorColumns + `
		FROM updated_post AS post
		` + postAuthorJoin

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
	return post, normalizePostError(err)
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
		post_text           pgtype.Text
		author_profile_id   pgtype.Int8
		author_community_id pgtype.Int8
		createdAt           pgtype.Timestamptz
		updatedAt           pgtype.Timestamptz
		deletedAt           pgtype.Timestamptz

		authorName pgtype.Text
		avatarURL  pgtype.Text
		authorType pgtype.Text
	)

	dest := []any{
		&id,
		&post_text,
		&author_profile_id,
		&author_community_id,
		&createdAt,
		&updatedAt,
		&deletedAt,
		&authorName,
		&avatarURL,
		&authorType,
	}

	if err := row.Scan(dest...); err != nil {
		return domain.Post{}, err
	}

	post := domain.Post{
		ID:                id,
		PostText:          textPointer(post_text),
		AuthorProfileID:   int64Pointer(author_profile_id),
		AuthorCommunityID: int64Pointer(author_community_id),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         timestampPointer(updatedAt),
		DeletedAt:         timestampPointer(deletedAt),
	}

	post.Author = domain.Author{
		Name: authorName.String,
		Type: domain.PostAuthorType(authorType.String),
	}

	if avatarURL.Valid {
		post.Author.AvatarURL = avatarURL.String
	}

	return post, nil
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
