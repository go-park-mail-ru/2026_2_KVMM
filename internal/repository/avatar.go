package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxProfileAvatars = 10

var (
	ErrAvatarLimit    = errors.New("avatar limit reached") // уже загрузил максимум аватарок (10)
	ErrAvatarNotFound = errors.New("avatar not found")
	ErrInvalidAvatar  = errors.New("invalid avatar image")
)

// AvatarRepository хранит файлы аватарок и связи с профилями
type AvatarRepository struct {
	pool     dbPool
	mediaDir string
	mediaURL string
}

// NewAvatarRepository создаёт репозиторий аватарок
func NewAvatarRepository(pool dbPool, mediaDir, mediaURL string) *AvatarRepository {
	if strings.TrimSpace(mediaDir) == "" {
		mediaDir = "./internal/store/media"
	}
	if strings.TrimSpace(mediaURL) == "" {
		mediaURL = "/api/media"
	}
	return &AvatarRepository{
		pool:     pool,
		mediaDir: mediaDir,
		mediaURL: strings.TrimRight(mediaURL, "/"),
	}
}

const avatarLockQuery = `
	SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`

const avatarProfileExistsQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM profile
		WHERE id = $1
		  AND deleted_at IS NULL
	)`

const avatarCountQuery = `
	SELECT COUNT(*)
	FROM profile_avatar pa
	JOIN media m ON m.id = pa.media_id
	WHERE pa.profile_id = $1
	  AND pa.deleted_at IS NULL
	  AND m.deleted_at IS NULL`

const createMediaQuery = `
	INSERT INTO media (media_path)
	VALUES ($1)
	RETURNING id, created_at`

const createProfileAvatarQuery = `
	INSERT INTO profile_avatar (profile_id, media_id)
	VALUES ($1, $2)`

const listAvatarsQuery = `
	SELECT m.id, m.media_path, pa.created_at
	FROM profile_avatar pa
	JOIN media m ON m.id = pa.media_id
	WHERE pa.profile_id = $1
	  AND pa.deleted_at IS NULL
	  AND m.deleted_at IS NULL
	ORDER BY pa.created_at DESC, m.id DESC`

const readAvatarForDeleteQuery = `
	SELECT m.media_path
	FROM profile_avatar pa
	JOIN media m ON m.id = pa.media_id
	WHERE pa.profile_id = $1
	  AND pa.media_id = $2
	  AND pa.deleted_at IS NULL
	  AND m.deleted_at IS NULL
	FOR UPDATE OF pa, m`

const deleteProfileAvatarQuery = `
	UPDATE profile_avatar
	SET deleted_at = CURRENT_TIMESTAMP
	WHERE profile_id = $1
	  AND media_id = $2
	  AND deleted_at IS NULL`

// Create сохраняет новую аватарку и добавляет её к профилю.
func (r *AvatarRepository) Create(
	ctx context.Context,
	profileID int64,
	req domain.UploadAvatarRequest,
) (domain.Avatar, error) {
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
	relativePath := filepath.ToSlash(filepath.Join("avatars", fileName))
	localPath := filepath.Join(r.mediaDir, filepath.FromSlash(relativePath))
	publicPath := r.mediaURL + "/" + relativePath

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return domain.Avatar{}, fmt.Errorf("create avatar directory: %w", err)
	}
	if err := os.WriteFile(localPath, req.Data, 0o644); err != nil {
		return domain.Avatar{}, fmt.Errorf("save avatar file: %w", err)
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

	lockKey := fmt.Sprintf("profile-avatar:%d", profileID)
	if _, err = tx.Exec(ctx, avatarLockQuery, lockKey); err != nil {
		return domain.Avatar{}, err
	}

	var profileExists bool
	if err = tx.QueryRow(ctx, avatarProfileExistsQuery, profileID).Scan(&profileExists); err != nil {
		return domain.Avatar{}, err
	}
	if !profileExists {
		return domain.Avatar{}, ErrProfileNotFound
	}

	var avatarCount int
	if err = tx.QueryRow(ctx, avatarCountQuery, profileID).Scan(&avatarCount); err != nil {
		return domain.Avatar{}, err
	}
	if avatarCount >= maxProfileAvatars {
		return domain.Avatar{}, ErrAvatarLimit
	}

	var avatar domain.Avatar
	if err = tx.QueryRow(ctx, createMediaQuery, publicPath).Scan(&avatar.ID, &avatar.CreatedAt); err != nil {
		return domain.Avatar{}, normalizeAvatarError(err)
	}
	if _, err = tx.Exec(ctx, createProfileAvatarQuery, profileID, avatar.ID); err != nil {
		return domain.Avatar{}, normalizeAvatarError(err)
	}

	if err = tx.Commit(ctx); err != nil {
		return domain.Avatar{}, err
	}
	removeFile = false
	avatar.URL = publicPath
	return avatar, nil
}

// List возвращает активные аватарки от самой новой к самой старой.
func (r *AvatarRepository) List(ctx context.Context, profileID int64) (domain.AvatarsResponse, error) {
	if profileID <= 0 {
		return domain.AvatarsResponse{}, ErrProfileNotFound
	}
	rows, err := r.pool.Query(ctx, listAvatarsQuery, profileID)
	if err != nil {
		return domain.AvatarsResponse{}, err
	}
	defer rows.Close()

	response := domain.AvatarsResponse{Avatars: make([]domain.Avatar, 0)}
	for rows.Next() {
		var (
			avatar    domain.Avatar
			mediaPath pgtype.Text
			createdAt pgtype.Timestamptz
		)
		if err := rows.Scan(&avatar.ID, &mediaPath, &createdAt); err != nil {
			return domain.AvatarsResponse{}, err
		}
		avatar.URL = mediaPath.String
		avatar.CreatedAt = createdAt.Time
		response.Avatars = append(response.Avatars, avatar)
	}
	if err := rows.Err(); err != nil {
		return domain.AvatarsResponse{}, err
	}
	return response, nil
}

// Delete удаляет аватарку только у владельца профиля.
func (r *AvatarRepository) Delete(ctx context.Context, profileID, avatarID int64) error {
	if profileID <= 0 || avatarID <= 0 {
		return ErrAvatarNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var mediaPath string
	if err := tx.QueryRow(ctx, readAvatarForDeleteQuery, profileID, avatarID).Scan(&mediaPath); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAvatarNotFound
		}
		return err
	}
	if _, err := tx.Exec(ctx, deleteProfileAvatarQuery, profileID, avatarID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if err := r.removeStoredFile(mediaPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove avatar file: %w", err)
	}
	return nil
}

func avatarExtension(data []byte) (string, bool) {
	switch http.DetectContentType(data) {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}

func randomAvatarName(extension string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value) + extension, nil
}

func (r *AvatarRepository) removeStoredFile(mediaPath string) error {
	prefix := r.mediaURL + "/"
	if !strings.HasPrefix(mediaPath, prefix) {
		return nil
	}
	relativePath := strings.TrimPrefix(mediaPath, prefix)
	if relativePath == "" || strings.Contains(relativePath, "..") {
		return nil
	}
	return os.Remove(filepath.Join(r.mediaDir, filepath.FromSlash(relativePath)))
}

func normalizeAvatarError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAvatarNotFound
	}
	if isUniqueViolation(err) {
		return ErrAvatarLimit
	}
	return err
}
