package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrCommunityNotFound       = errors.New("community not found")
	ErrCommunityForbidden      = errors.New("community permission denied")
	ErrCommunityMemberNotFound = errors.New("community membership not found")
	ErrCommunityConflict       = errors.New("community name already exists")
)

const communitySelectColumns = `
	c.id,
	c.uid,
	c.community_name,
	c.bio,
	c.created_at,
	c.updated_at,
	(SELECT COUNT(*) FROM community_member cm WHERE cm.community_id = c.id AND cm.deleted_at IS NULL),
	(SELECT m.media_path
	 FROM community_avatar ca
	 JOIN media m ON m.id = ca.media_id
	 WHERE ca.community_id = c.id AND ca.deleted_at IS NULL AND m.deleted_at IS NULL
	 ORDER BY ca.created_at DESC, m.id DESC
	 LIMIT 1)`

const createCommunityQuery = `
		INSERT INTO community (community_name, bio)
		VALUES ($1, $2)
		RETURNING id, uid, community_name, bio, created_at, updated_at, 0, NULL`

const createCommunityOwnerQuery = `
		INSERT INTO community_member (profile_id, community_id, role)
		VALUES ($1, $2, 'owner')`

const createCommunityAvatarAuthorizationQuery = `
		SELECT c.id
		FROM community c
		WHERE c.uid = $1::uuid AND c.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM community_member cm
				WHERE cm.community_id = c.id AND cm.profile_id = $2
					AND cm.deleted_at IS NULL AND cm.role IN ('owner', 'admin')
			)
		FOR UPDATE OF c`

const createCommunityAvatarQuery = `
		INSERT INTO community_avatar (community_id, media_id)
		VALUES ($1, $2)`

const communityAvatarFailureQuery = `
		SELECT EXISTS(
			SELECT 1 FROM community
			WHERE uid = $1::uuid AND deleted_at IS NULL
		)`

const readCommunitiesByIDsQuery = `
		SELECT ` + communitySelectColumns + `
		FROM community c
		WHERE c.id = ANY($1::BIGINT[]) AND c.deleted_at IS NULL`

const readCommunityIDByUIDQuery = `
		SELECT id
		FROM community
		WHERE uid = $1::uuid AND deleted_at IS NULL`

const readCommunityByUIDQuery = `
		SELECT ` + communitySelectColumns + `
		FROM community c
		WHERE c.uid = $1::uuid AND c.deleted_at IS NULL`

const listCommunitiesQuery = `
		SELECT ` + communitySelectColumns + `
		FROM community c
		WHERE c.deleted_at IS NULL
		ORDER BY c.id DESC OFFSET $1 LIMIT $2`

const listCommunitiesForSearchQuery = `
	SELECT ` + communitySelectColumns + `
	FROM community c
	WHERE c.deleted_at IS NULL
	ORDER BY c.id
	LIMIT $1 OFFSET $2`

const listCommunitySubscriptionsQuery = `
		SELECT ` + communitySelectColumns + `
		FROM community_member cm
		JOIN community c ON c.id = cm.community_id
		WHERE cm.profile_id = $1 AND cm.deleted_at IS NULL AND c.deleted_at IS NULL
		ORDER BY cm.created_at DESC, c.id DESC OFFSET $2 LIMIT $3`

const updateCommunityQuery = `
		UPDATE community c SET
			community_name = COALESCE($3, c.community_name),
			bio = COALESCE($4, c.bio),
			updated_at = NOW()
		WHERE c.uid = $1::uuid AND c.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM community_member cm
				WHERE cm.community_id = c.id AND cm.profile_id = $2
					AND cm.deleted_at IS NULL AND cm.role IN ('owner', 'admin')
			)
		RETURNING c.id, c.uid, c.community_name, c.bio, c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM community_member cm WHERE cm.community_id = c.id AND cm.deleted_at IS NULL),
			(SELECT m.media_path
			 FROM community_avatar ca
			 JOIN media m ON m.id = ca.media_id
			 WHERE ca.community_id = c.id AND ca.deleted_at IS NULL AND m.deleted_at IS NULL
			 ORDER BY ca.created_at DESC, m.id DESC
			 LIMIT 1)`

const deleteCommunityQuery = `
		UPDATE community c SET deleted_at = NOW(), updated_at = NOW()
		WHERE c.uid = $1::uuid AND c.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM community_member cm
				WHERE cm.community_id = c.id AND cm.profile_id = $2
					AND cm.deleted_at IS NULL AND cm.role = 'owner'
			)
		RETURNING c.id`

const deleteCommunityMembersQuery = `
		UPDATE community_member
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE community_id = $1 AND deleted_at IS NULL`

const deleteCommunityPostsQuery = `
		UPDATE post
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE author_community_id = $1 AND deleted_at IS NULL`

const communityExistsByUIDQuery = `
		SELECT EXISTS(
			SELECT 1 FROM community
			WHERE uid = $1::uuid AND deleted_at IS NULL
		)`

const lockCommunityByUIDQuery = `
		SELECT id
		FROM community
		WHERE uid = $1::uuid AND deleted_at IS NULL
		FOR UPDATE`

const upsertCommunityMemberQuery = `
		INSERT INTO community_member (profile_id, community_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (profile_id, community_id) DO UPDATE SET
			deleted_at = NULL,
			role = CASE
				WHEN community_member.deleted_at IS NULL THEN community_member.role
				ELSE 'member'::role_enum
			END,
			updated_at = NOW()`

const unsubscribeCommunityQuery = `
		UPDATE community_member cm
		SET deleted_at = NOW(), updated_at = NOW()
		FROM community c
		WHERE cm.community_id = c.id AND c.uid = $2::uuid AND c.deleted_at IS NULL
			AND cm.profile_id = $1 AND cm.deleted_at IS NULL AND cm.role <> 'owner'`

const assignCommunityAdminQuery = `
		UPDATE community_member target
		SET role = 'admin', updated_at = NOW()
		FROM community c
		WHERE target.community_id = c.id AND c.uid = $1::uuid AND c.deleted_at IS NULL
			AND target.profile_id = $2 AND target.deleted_at IS NULL AND target.role <> 'owner'
			AND EXISTS (
				SELECT 1 FROM community_member owner
				WHERE owner.community_id = c.id AND owner.profile_id = $3
					AND owner.role = 'owner' AND owner.deleted_at IS NULL
			)`

const authorizeCommunityPostWriteQuery = `
		SELECT c.id, c.community_name
		FROM community c
		JOIN community_member cm ON cm.community_id = c.id
		WHERE c.uid = $1::uuid AND c.deleted_at IS NULL
			AND cm.profile_id = $2 AND cm.deleted_at IS NULL
			AND cm.role IN ('owner', 'admin', 'moderator')`

// CommunityRepository выполняет SQL-операции для сообществ и их участников
type CommunityRepository struct {
	pool     dbPool
	mediaDir string
	mediaURL string
}

// NewCommunityRepository создаёт репозиторий сообществ
func NewCommunityRepository(pool dbPool, media ...string) *CommunityRepository {
	mediaDir, mediaURL := "./internal/store/media", "/api/media"
	if len(media) > 0 && strings.TrimSpace(media[0]) != "" {
		mediaDir = media[0]
	}
	if len(media) > 1 && strings.TrimSpace(media[1]) != "" {
		mediaURL = strings.TrimRight(media[1], "/")
	}
	return &CommunityRepository{pool: pool, mediaDir: mediaDir, mediaURL: mediaURL}
}

// Create создаёт сообщество и назначает создателя владельцем в одной транзакции
func (r *CommunityRepository) Create(ctx context.Context, ownerID int64, req domain.CreateCommunityRequest) (domain.Community, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Community{}, err
	}
	defer tx.Rollback(ctx)
	community, err := scanCommunity(tx.QueryRow(ctx, createCommunityQuery, strings.TrimSpace(req.Name), strings.TrimSpace(req.Bio)))
	if err != nil {
		return domain.Community{}, normalizeCommunityError(err)
	}
	if _, err := tx.Exec(ctx, createCommunityOwnerQuery, ownerID, community.ID); err != nil {
		return domain.Community{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Community{}, err
	}
	community.SubscribersCount = 1
	return community, nil
}

// CreateAvatar сохраняет изображение и назначает его аватаром сообщества.
func (r *CommunityRepository) CreateAvatar(ctx context.Context, profileID int64, communityUID string, req domain.UploadAvatarRequest) (domain.Avatar, error) {
	if profileID <= 0 || len(req.Data) == 0 || len(req.Data) > domain.MaxAvatarSize {
		return domain.Avatar{}, ErrInvalidAvatar
	}
	extension, ok := avatarExtension(req.Data)
	if !ok {
		return domain.Avatar{}, ErrInvalidAvatar
	}
	fileName, err := randomAvatarName(extension)
	if err != nil {
		return domain.Avatar{}, err
	}
	relativePath := filepath.ToSlash(filepath.Join("community_avatars", fileName))
	localPath := filepath.Join(r.mediaDir, filepath.FromSlash(relativePath))
	publicPath := r.mediaURL + "/" + relativePath
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return domain.Avatar{}, fmt.Errorf("create community avatar directory: %w", err)
	}
	if err := os.WriteFile(localPath, req.Data, 0o644); err != nil {
		return domain.Avatar{}, fmt.Errorf("save community avatar file: %w", err)
	}
	removeFile := true
	defer func() {
		if removeFile {
			_ = os.Remove(localPath)
		}
	}()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Avatar{}, err
	}
	defer tx.Rollback(ctx)
	var communityID int64
	if err := tx.QueryRow(ctx, createCommunityAvatarAuthorizationQuery, communityUID, profileID).Scan(&communityID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if lookupErr := r.pool.QueryRow(ctx, communityAvatarFailureQuery, communityUID).Scan(&exists); lookupErr != nil {
				return domain.Avatar{}, lookupErr
			}
			if !exists {
				return domain.Avatar{}, ErrCommunityNotFound
			}
			return domain.Avatar{}, ErrCommunityForbidden
		}
		return domain.Avatar{}, err
	}

	var avatar domain.Avatar
	var uid pgtype.UUID
	if err := tx.QueryRow(ctx, createMediaQuery, publicPath).Scan(&avatar.ID, &uid, &avatar.CreatedAt); err != nil {
		return domain.Avatar{}, err
	}
	if _, err := tx.Exec(ctx, createCommunityAvatarQuery, communityID, avatar.ID); err != nil {
		return domain.Avatar{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Avatar{}, err
	}
	removeFile = false
	avatar.UID = uuidString(uid)
	avatar.URL = publicPath
	return avatar, nil
}

// ReadByIDs пакетно читает сообщества по внутренним ID для Posts service
func (r *CommunityRepository) ReadByIDs(ctx context.Context, ids []int64) ([]domain.Community, error) {
	if len(ids) == 0 {
		return []domain.Community{}, nil
	}
	rows, err := r.pool.Query(ctx, readCommunitiesByIDsQuery, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommunities(rows)
}

// CommunityIDByUID возращает ID по UID
func (r *CommunityRepository) CommunityIDByUID(ctx context.Context, uid string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, readCommunityIDByUIDQuery, uid).Scan(&id)
	if err != nil {
		return 0, normalizeCommunityError(err)
	}
	return id, nil
}

// Read возвращает активное сообщество по UUID
func (r *CommunityRepository) Read(ctx context.Context, uid string) (domain.Community, error) {
	community, err := scanCommunity(r.pool.QueryRow(ctx, readCommunityByUIDQuery, uid))
	return community, normalizeCommunityError(err)
}

// List возвращает страницу активных сообществ
func (r *CommunityRepository) List(ctx context.Context, offset, limit int) (domain.CommunityPage, error) {
	rows, err := r.pool.Query(ctx, listCommunitiesQuery, offset, limit+1)
	if err != nil {
		return domain.CommunityPage{}, err
	}
	defer rows.Close()
	items, err := scanCommunities(rows)
	if err != nil {
		return domain.CommunityPage{}, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return domain.CommunityPage{Communities: items, Offset: offset, Limit: limit, HasMore: more}, nil
}

// ListForSearch возвращает страницу активных сообществ для пересборки поискового индекса
func (r *CommunityRepository) ListForSearch(ctx context.Context, offset, limit int) ([]domain.Community, error) {
	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("invalid search page: offset=%d limit=%d", offset, limit)
	}
	rows, err := r.pool.Query(ctx, listCommunitiesForSearchQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommunities(rows)
}

// ListSubscriptions возвращает сообщества, на которые подписан профиль
func (r *CommunityRepository) ListSubscriptions(ctx context.Context, profileID int64, offset, limit int) (domain.CommunityPage, error) {
	rows, err := r.pool.Query(ctx, listCommunitySubscriptionsQuery, profileID, offset, limit+1)
	if err != nil {
		return domain.CommunityPage{}, err
	}
	defer rows.Close()
	items, err := scanCommunities(rows)
	if err != nil {
		return domain.CommunityPage{}, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return domain.CommunityPage{Communities: items, Offset: offset, Limit: limit, HasMore: more}, nil
}

// Update изменяет сообщество, если запрос делает его владелец или администратор
func (r *CommunityRepository) Update(ctx context.Context, actorID int64, uid string, req domain.UpdateCommunityRequest) (domain.Community, error) {
	var name, bio any
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Bio != nil {
		bio = strings.TrimSpace(*req.Bio)
	}
	community, err := scanCommunity(r.pool.QueryRow(ctx, updateCommunityQuery, uid, actorID, name, bio))
	if err != nil {
		return domain.Community{}, normalizeCommunityError(err)
	}
	return community, nil
}

// Delete мягко удаляет сообщество, его участников и публикации транзакционно
func (r *CommunityRepository) Delete(ctx context.Context, actorID int64, uid string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var communityID int64
	err = tx.QueryRow(ctx, deleteCommunityQuery, uid, actorID).Scan(&communityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.communityDeleteFailure(ctx, uid)
		}
		return err
	}
	if _, err := tx.Exec(ctx, deleteCommunityMembersQuery, communityID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, deleteCommunityPostsQuery, communityID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CommunityRepository) communityDeleteFailure(ctx context.Context, uid string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, communityExistsByUIDQuery, uid).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrCommunityNotFound
	}
	return ErrCommunityForbidden
}

// Subscribe добавляет участника или восстанавливает ранее удалённую подписку
func (r *CommunityRepository) Subscribe(ctx context.Context, profileID int64, uid string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var communityID int64
	if err := tx.QueryRow(ctx, lockCommunityByUIDQuery, uid).Scan(&communityID); err != nil {
		return normalizeCommunityError(err)
	}
	if _, err := tx.Exec(ctx, upsertCommunityMemberQuery, profileID, communityID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Unsubscribe удаляет подписку участника, не позволяя владельцу покинуть своё сообщество
func (r *CommunityRepository) Unsubscribe(ctx context.Context, profileID int64, uid string) error {
	result, err := r.pool.Exec(ctx, unsubscribeCommunityQuery, profileID, uid)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCommunityMemberNotFound
	}
	return nil
}

// AssignAdmin назначает администратором уже подписанного участника (вызвать может только владелец)
func (r *CommunityRepository) AssignAdmin(ctx context.Context, ownerID, profileID int64, uid string) error {
	result, err := r.pool.Exec(ctx, assignCommunityAdminQuery, uid, profileID, ownerID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCommunityForbidden
	}
	return nil
}

// AuthorizePostWrite возвращает сообщество, если пользователь может публиковать от его имени
func (r *CommunityRepository) AuthorizePostWrite(ctx context.Context, profileID int64, uid string) (int64, string, error) {
	var communityID int64
	var communityName string
	err := r.pool.QueryRow(ctx, authorizeCommunityPostWriteQuery, uid, profileID).Scan(&communityID, &communityName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrCommunityForbidden
		}
		return 0, "", err
	}
	return communityID, communityName, nil
}

func scanCommunity(row rowScanner) (domain.Community, error) {
	var id int64
	var uid pgtype.UUID
	var name, bio pgtype.Text
	var created, updated pgtype.Timestamptz
	var avatarURL pgtype.Text
	var subscribers int64
	if err := row.Scan(&id, &uid, &name, &bio, &created, &updated, &subscribers, &avatarURL); err != nil {
		return domain.Community{}, err
	}
	community := domain.Community{ID: id, UID: uuidString(uid), Name: name.String, Bio: bio.String, CreatedAt: created.Time, UpdatedAt: timestampPointer(updated), SubscribersCount: subscribers}
	if avatarURL.Valid {
		community.AvatarURL = &avatarURL.String
	}
	return community, nil
}

func scanCommunities(rows pgx.Rows) ([]domain.Community, error) {
	items := make([]domain.Community, 0)
	for rows.Next() {
		item, err := scanCommunity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func normalizeCommunityError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCommunityNotFound
	}
	if isUniqueViolation(err) {
		return ErrCommunityConflict
	}
	return err
}
