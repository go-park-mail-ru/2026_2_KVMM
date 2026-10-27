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
	ErrRelationshipNotFound = errors.New("relationship not found")
	ErrRelationshipConflict = errors.New("profiles already have an active relationship")
	ErrInvalidRelationship  = errors.New("invalid relationship")
)

// RelationshipRepository выполняет SQL-операции со связями профилей
type RelationshipRepository struct {
	pool dbPool
}

// NewRelationshipRepository создаёт репозиторий связей
func NewRelationshipRepository(pool dbPool) *RelationshipRepository {
	return &RelationshipRepository{pool: pool}
}

const relationshipColumns = `
	profile_id_master,
	profile_id_slave,
	relationships_status::TEXT,
	relationships_type,
	created_at,
	updated_at,
	deleted_at`

const relationshipPairLockQuery = `
	SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`

const relationshipPairQuery = `
	SELECT ` + relationshipColumns + `
	FROM profile_relationship
	WHERE (profile_id_master = $1 AND profile_id_slave = $2)
	   OR (profile_id_master = $2 AND profile_id_slave = $1)
	ORDER BY deleted_at NULLS FIRST, created_at DESC
	FOR UPDATE`

const createRelationshipQuery = `
	INSERT INTO profile_relationship (
		profile_id_master,
		profile_id_slave,
		relationships_status,
		relationships_type
	)
	VALUES ($1, $2, 'request', $3)
	RETURNING ` + relationshipColumns

const updateRelationshipRequestQuery = `
	UPDATE profile_relationship
	SET relationships_status = 'request',
		relationships_type = $3,
		created_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP,
		deleted_at = NULL
	WHERE profile_id_master = $1
	  AND profile_id_slave = $2
	RETURNING ` + relationshipColumns

const archiveRelationshipPairQuery = `
	UPDATE profile_relationship
	SET deleted_at = CURRENT_TIMESTAMP,
		updated_at = CURRENT_TIMESTAMP
	WHERE ((profile_id_master = $1 AND profile_id_slave = $2)
	    OR (profile_id_master = $2 AND profile_id_slave = $1))
		AND deleted_at IS NULL`

const listRelationshipsQuery = `
	SELECT ` + relationshipColumns + `
	FROM profile_relationship
	WHERE deleted_at IS NULL
		AND (profile_id_master = $1 OR profile_id_slave = $1)
		AND ($2 = '' OR relationships_status::TEXT = $2)
		AND ($3 = '' OR relationships_type = $3)
	ORDER BY created_at DESC, profile_id_master DESC, profile_id_slave DESC
	OFFSET $4
	LIMIT $5`

const listIncomingRequestsQuery = `
	SELECT ` + relationshipColumns + `
	FROM profile_relationship
	WHERE deleted_at IS NULL
		AND relationships_status = 'request'
		AND profile_id_slave = $1
		AND ($2 = '' OR relationships_type = $2)
	ORDER BY created_at DESC, profile_id_master DESC
	OFFSET $3
	LIMIT $4`

const listTypesQuery = `
	SELECT DISTINCT relationships_type
	FROM profile_relationship
	WHERE deleted_at IS NULL
		AND relationships_status <> 'canceled'
		AND (profile_id_master = $1 OR profile_id_slave = $1)
	ORDER BY relationships_type`

const myConnectionsQuery = `
	SELECT DISTINCT CASE
		WHEN profile_id_master = $1 THEN profile_id_slave
		ELSE profile_id_master
	END AS profile_id
	FROM profile_relationship
	WHERE (profile_id_master = $1 OR profile_id_slave = $1)
		AND relationships_status = 'accepted'
		AND deleted_at IS NULL`

const suggestionCandidatesQuery = `
	WITH connection_ids AS (
		SELECT DISTINCT unnest($1::BIGINT[]) AS profile_id
	)
	SELECT CASE
		WHEN r.profile_id_master = c.profile_id THEN r.profile_id_slave
		ELSE r.profile_id_master
	END AS profile_id,
	COUNT(DISTINCT c.profile_id) AS common_count
	FROM connection_ids c
	JOIN profile_relationship r
		ON r.profile_id_master = c.profile_id
		OR r.profile_id_slave = c.profile_id
	WHERE r.relationships_status = 'accepted'
		AND r.deleted_at IS NULL
	GROUP BY 1
	ORDER BY common_count DESC, profile_id ASC`

const shortProfilesQuery = `
	SELECT
		p.id,
		p.nickname,
		p.profile_name,
		p.surname,
		p.patronymic,
		avatar.media_path AS avatar_url
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
	WHERE p.id = ANY($1::BIGINT[])
		AND p.deleted_at IS NULL`

const directRelationshipProfilesQuery = `
	SELECT CASE
		WHEN profile_id_master = $1 THEN profile_id_slave
		ELSE profile_id_master
	END AS profile_id
	FROM profile_relationship
	WHERE deleted_at IS NULL
		AND relationships_status <> 'canceled'
		AND (
			(profile_id_master = $1 AND profile_id_slave = ANY($2::BIGINT[]))
			OR (profile_id_slave = $1 AND profile_id_master = ANY($2::BIGINT[]))
		)`

const acceptRelationshipQuery = `
	UPDATE profile_relationship
	SET relationships_status = 'accepted',
		updated_at = CURRENT_TIMESTAMP
	WHERE profile_id_master = $1
		AND profile_id_slave = $2
		AND relationships_type = $3
		AND relationships_status = 'request'
		AND deleted_at IS NULL
	RETURNING ` + relationshipColumns

const cancelRelationshipQuery = `
	UPDATE profile_relationship
	SET relationships_status = 'canceled',
		updated_at = CURRENT_TIMESTAMP
	WHERE deleted_at IS NULL
		AND relationships_status <> 'canceled'
		AND relationships_type = $3
		AND (
			(profile_id_master = $1 AND profile_id_slave = $2)
			OR (profile_id_master = $2 AND profile_id_slave = $1)
		)
	RETURNING ` + relationshipColumns

type suggestionCandidate struct {
	ProfileID   int64
	CommonCount int64
}

// Create создаёт запрос на новую связь
func (r *RelationshipRepository) Create(
	ctx context.Context,
	masterID int64,
	req domain.CreateRelationshipRequest,
) (domain.ProfileRelationship, error) {
	typeName := strings.TrimSpace(req.RelationshipsType)
	if masterID <= 0 || req.ProfileIDSlave <= 0 || masterID == req.ProfileIDSlave || typeName == "" || len([]rune(typeName)) > 32 {
		return domain.ProfileRelationship{}, ErrInvalidRelationship
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ProfileRelationship{}, err
	}
	defer tx.Rollback(ctx)

	lockKey := fmt.Sprintf("profile-relationship:%d:%d", minID(masterID, req.ProfileIDSlave), maxID(masterID, req.ProfileIDSlave))
	if _, err = tx.Exec(ctx, relationshipPairLockQuery, lockKey); err != nil {
		return domain.ProfileRelationship{}, err
	}

	const checkAbleToRel = `SELECT COUNT(*) FROM profile WHERE id = ANY($1::BIGINT[]) AND deleted_at IS NULL`
	var profileCount int
	if err = tx.QueryRow(ctx,
		checkAbleToRel,
		[]int64{masterID, req.ProfileIDSlave},
	).Scan(&profileCount); err != nil {
		return domain.ProfileRelationship{}, err
	}
	if profileCount != 2 {
		return domain.ProfileRelationship{}, ErrProfileNotFound
	}

	rows, err := tx.Query(ctx, relationshipPairQuery, masterID, req.ProfileIDSlave)
	if err != nil {
		return domain.ProfileRelationship{}, err
	}
	var existing []domain.ProfileRelationship
	for rows.Next() {
		relationship, scanErr := scanRelationship(rows)
		if scanErr != nil {
			rows.Close()
			return domain.ProfileRelationship{}, scanErr
		}
		existing = append(existing, relationship)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return domain.ProfileRelationship{}, err
	}
	rows.Close()

	for _, relationship := range existing {
		if relationship.DeletedAt == nil && relationship.RelationshipsStatus != domain.RelationshipCanceled {
			return domain.ProfileRelationship{}, ErrRelationshipConflict
		}
	}

	var created domain.ProfileRelationship
	for _, relationship := range existing {
		if relationship.ProfileIDMaster == masterID && relationship.ProfileIDSlave == req.ProfileIDSlave {
			created, err = scanRelationship(tx.QueryRow(ctx, updateRelationshipRequestQuery, masterID, req.ProfileIDSlave, typeName))
			if err != nil {
				return domain.ProfileRelationship{}, err
			}
			break
		}
	}

	if created.ProfileIDMaster == 0 {
		if _, err = tx.Exec(ctx, archiveRelationshipPairQuery, masterID, req.ProfileIDSlave); err != nil {
			return domain.ProfileRelationship{}, err
		}
		created, err = scanRelationship(tx.QueryRow(ctx, createRelationshipQuery, masterID, req.ProfileIDSlave, typeName))
		if err != nil {
			return domain.ProfileRelationship{}, normalizeRelationshipError(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ProfileRelationship{}, err
	}
	if err := r.attachRelationshipProfile(ctx, &created, masterID); err != nil {
		return domain.ProfileRelationship{}, err
	}
	return created, nil
}

// List возвращает связи пользователя с фильтрацией и пагинацией
func (r *RelationshipRepository) List(
	ctx context.Context,
	profileID int64,
	req domain.ListRelationshipsRequest,
) (domain.RelationshipsResponse, error) {
	return r.list(ctx, listRelationshipsQuery, profileID, string(req.RelationshipsStatus), strings.TrimSpace(req.RelationshipsType), req.Cursor, req.Limit)
}

// Requests возвращает входящие запросы пользователя
func (r *RelationshipRepository) Requests(
	ctx context.Context,
	profileID int64,
	typeName string,
	cursor int64,
	limit int,
) (domain.RelationshipsResponse, error) {
	return r.list(ctx, listIncomingRequestsQuery, profileID, "", strings.TrimSpace(typeName), cursor, limit)
}

// Types возвращает виды активных связей пользователя
func (r *RelationshipRepository) Types(ctx context.Context, profileID int64) (domain.RelationshipTypes, error) {
	if profileID <= 0 {
		return domain.RelationshipTypes{}, ErrProfileNotFound
	}
	rows, err := r.pool.Query(ctx, listTypesQuery, profileID)
	if err != nil {
		return domain.RelationshipTypes{}, err
	}
	defer rows.Close()

	result := domain.RelationshipTypes{RelationshipsTypes: make([]string, 0)}
	for rows.Next() {
		var typeName string
		if err := rows.Scan(&typeName); err != nil {
			return domain.RelationshipTypes{}, err
		}
		result.RelationshipsTypes = append(result.RelationshipsTypes, typeName)
	}
	if err := rows.Err(); err != nil {
		return domain.RelationshipTypes{}, err
	}
	return result, nil
}

// Suggestions возвращает "друзей друзей", отсортированных по числу общих связей
func (r *RelationshipRepository) Suggestions(
	ctx context.Context,
	profileID int64,
	cursor int64,
	limit int,
) (domain.RelationshipSuggestionsResponse, error) {
	if profileID <= 0 {
		return domain.RelationshipSuggestionsResponse{}, ErrProfileNotFound
	}
	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 10)

	response := domain.RelationshipSuggestionsResponse{
		Profiles: make([]domain.RelationshipSuggestion, 0),
		Cursor:   cursor,
		Limit:    limit,
	}

	connections, err := r.findMyConnectionIDs(ctx, profileID)
	if err != nil {
		return response, err
	}
	if len(connections) == 0 {
		return response, nil
	}

	candidates, err := r.findSuggestionCandidates(ctx, connections)
	if err != nil {
		return response, err
	}
	if len(candidates) == 0 {
		return response, nil
	}

	candidateIDs := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.ProfileID)
	}

	profiles, err := r.loadShortProfiles(ctx, candidateIDs)
	if err != nil {
		return response, err
	}
	directIDs, err := r.findDirectRelationshipProfiles(ctx, profileID, candidateIDs)
	if err != nil {
		return response, err
	}

	filtered := make([]domain.RelationshipSuggestion, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ProfileID == profileID || directIDs[candidate.ProfileID] {
			continue
		}
		profile, exists := profiles[candidate.ProfileID]
		if !exists {
			continue
		}
		filtered = append(filtered, domain.RelationshipSuggestion{
			Profile:     profile,
			CommonCount: candidate.CommonCount,
		})
	}

	page := suggestionPage(filtered, cursor, limit)
	response.Profiles, response.NextCursor, response.HasMore = paginateSuggestions(page, limit, cursor)
	return response, nil
}

// findMyConnectionIDs возвращает id подтверждённых связей пользователя
func (r *RelationshipRepository) findMyConnectionIDs(ctx context.Context, profileID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, myConnectionsQuery, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connections := make([]int64, 0)
	for rows.Next() {
		var connectionID int64
		if err := rows.Scan(&connectionID); err != nil {
			return nil, err
		}
		connections = append(connections, connectionID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return connections, nil
}

// findSuggestionCandidates возвращает пользователей и количество общих связей с ними
func (r *RelationshipRepository) findSuggestionCandidates(ctx context.Context, connections []int64) ([]suggestionCandidate, error) {
	rows, err := r.pool.Query(ctx, suggestionCandidatesQuery, connections)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]suggestionCandidate, 0)
	for rows.Next() {
		var candidate suggestionCandidate
		if err := rows.Scan(&candidate.ProfileID, &candidate.CommonCount); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

// loadShortProfiles загружает краткие данные профилей одним запросом
func (r *RelationshipRepository) loadShortProfiles(ctx context.Context, profileIDs []int64) (map[int64]domain.ShortProfile, error) {
	rows, err := r.pool.Query(ctx, shortProfilesQuery, profileIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make(map[int64]domain.ShortProfile, len(profileIDs))
	for rows.Next() {
		var (
			profile     domain.ShortProfile
			profileName pgtype.Text
			surname     pgtype.Text
			patronymic  pgtype.Text
			avatarURL   pgtype.Text
		)
		if err := rows.Scan(
			&profile.ID,
			&profile.Nickname,
			&profileName,
			&surname,
			&patronymic,
			&avatarURL,
		); err != nil {
			return nil, err
		}
		profile.ProfileName = textPointer(profileName)
		profile.Surname = textPointer(surname)
		profile.Patronymic = textPointer(patronymic)
		profile.AvatarURL = textPointer(avatarURL)
		profiles[profile.ID] = profile
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

// findDirectRelationshipProfiles возвращает профили, с которыми уже есть активная связь
func (r *RelationshipRepository) findDirectRelationshipProfiles(ctx context.Context, profileID int64, candidateIDs []int64) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, directRelationshipProfilesQuery, profileID, candidateIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	directIDs := make(map[int64]bool)
	for rows.Next() {
		var directID int64
		if err := rows.Scan(&directID); err != nil {
			return nil, err
		}
		directIDs[directID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return directIDs, nil
}

// suggestionPage оставляет текущую страницу и один дополнительный элемент для проверки HasMore
func suggestionPage(items []domain.RelationshipSuggestion, cursor int64, limit int) []domain.RelationshipSuggestion {
	if cursor >= int64(len(items)) {
		return items[:0]
	}
	start := int(cursor)
	end := start + limit + 1
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// Accept подтверждает входящий запрос на связь
func (r *RelationshipRepository) Accept(
	ctx context.Context,
	profileID int64,
	req domain.AcceptRelationshipRequest,
) (domain.ProfileRelationship, error) {
	if profileID <= 0 || req.ProfileID <= 0 || strings.TrimSpace(req.RelationshipsType) == "" {
		return domain.ProfileRelationship{}, ErrInvalidRelationship
	}
	relationship, err := scanRelationship(r.pool.QueryRow(
		ctx,
		acceptRelationshipQuery,
		req.ProfileID,
		profileID,
		strings.TrimSpace(req.RelationshipsType),
	))
	if err != nil {
		return domain.ProfileRelationship{}, normalizeRelationshipError(err)
	}
	if err := r.attachRelationshipProfile(ctx, &relationship, profileID); err != nil {
		return domain.ProfileRelationship{}, err
	}
	return relationship, nil
}

// Cancel отменяет исходящий запрос или закрывает активную связь
func (r *RelationshipRepository) Cancel(
	ctx context.Context,
	profileID int64,
	req domain.CancleRelationshipRequest,
) (domain.ProfileRelationship, error) {
	if profileID <= 0 || req.ProfileID <= 0 || strings.TrimSpace(req.RelationshipsType) == "" {
		return domain.ProfileRelationship{}, ErrInvalidRelationship
	}
	relationship, err := scanRelationship(r.pool.QueryRow(
		ctx,
		cancelRelationshipQuery,
		profileID,
		req.ProfileID,
		strings.TrimSpace(req.RelationshipsType),
	))
	if err != nil {
		return domain.ProfileRelationship{}, normalizeRelationshipError(err)
	}
	if err := r.attachRelationshipProfile(ctx, &relationship, profileID); err != nil {
		return domain.ProfileRelationship{}, err
	}
	return relationship, nil
}

func (r *RelationshipRepository) list(
	ctx context.Context,
	query string,
	profileID int64,
	firstFilter string,
	secondFilter string,
	cursor int64,
	limit int,
) (domain.RelationshipsResponse, error) {
	if profileID <= 0 {
		return domain.RelationshipsResponse{}, ErrProfileNotFound
	}
	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 10)
	response := domain.RelationshipsResponse{
		Relationships: make([]domain.ProfileRelationship, 0),
		Cursor:        cursor,
		Limit:         limit,
	}

	var rows pgx.Rows
	var err error
	if query == listIncomingRequestsQuery {
		rows, err = r.pool.Query(ctx, query, profileID, secondFilter, cursor, limit+1)
	} else {
		rows, err = r.pool.Query(ctx, query, profileID, firstFilter, secondFilter, cursor, limit+1)
	}
	if err != nil {
		return response, err
	}
	defer rows.Close()

	for rows.Next() {
		relationship, scanErr := scanRelationship(rows)
		if scanErr != nil {
			return response, scanErr
		}
		response.Relationships = append(response.Relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return response, err
	}
	response.Relationships, response.NextCursor, response.HasMore = paginateRelationships(
		response.Relationships,
		limit,
		cursor,
	)
	if err := r.attachRelationshipsProfiles(ctx, response.Relationships, profileID); err != nil {
		return response, err
	}
	return response, nil
}

// attachRelationshipProfile добавляет профиль второй стороны одной связи
func (r *RelationshipRepository) attachRelationshipProfile(
	ctx context.Context,
	relationship *domain.ProfileRelationship,
	currentProfileID int64,
) error {
	otherProfileID := relationship.ProfileIDMaster
	if relationship.ProfileIDMaster == currentProfileID {
		otherProfileID = relationship.ProfileIDSlave
	}
	profiles, err := r.loadShortProfiles(ctx, []int64{otherProfileID})
	if err != nil {
		return err
	}
	profile, exists := profiles[otherProfileID]
	if !exists {
		return ErrProfileNotFound
	}
	relationship.Profile = profile
	return nil
}

// attachRelationshipsProfiles добавляет профили вторых сторон страницы связей одним запросом
func (r *RelationshipRepository) attachRelationshipsProfiles(
	ctx context.Context,
	relationships []domain.ProfileRelationship,
	currentProfileID int64,
) error {
	if len(relationships) == 0 {
		return nil
	}

	profileIDs := make([]int64, 0, len(relationships))
	for _, relationship := range relationships {
		otherProfileID := relationship.ProfileIDMaster
		if relationship.ProfileIDMaster == currentProfileID {
			otherProfileID = relationship.ProfileIDSlave
		}
		profileIDs = append(profileIDs, otherProfileID)
	}
	profiles, err := r.loadShortProfiles(ctx, profileIDs)
	if err != nil {
		return err
	}

	for index := range relationships {
		otherProfileID := relationships[index].ProfileIDMaster
		if relationships[index].ProfileIDMaster == currentProfileID {
			otherProfileID = relationships[index].ProfileIDSlave
		}
		profile, exists := profiles[otherProfileID]
		if !exists {
			return ErrProfileNotFound
		}
		relationships[index].Profile = profile
	}
	return nil
}

func scanRelationship(row rowScanner) (domain.ProfileRelationship, error) {
	var (
		relationship domain.ProfileRelationship
		status       pgtype.Text
		typeName     pgtype.Text
		createdAt    pgtype.Timestamptz
		updatedAt    pgtype.Timestamptz
		deletedAt    pgtype.Timestamptz
	)
	if err := row.Scan(
		&relationship.ProfileIDMaster,
		&relationship.ProfileIDSlave,
		&status,
		&typeName,
		&createdAt,
		&updatedAt,
		&deletedAt,
	); err != nil {
		return domain.ProfileRelationship{}, err
	}
	relationship.RelationshipsStatus = domain.RelationshipStatus(status.String)
	relationship.RelationshipsType = typeName.String
	relationship.CreatedAt = createdAt.Time
	relationship.UpdatedAt = timestampPointer(updatedAt)
	relationship.DeletedAt = timestampPointer(deletedAt)
	return relationship, nil
}

func paginateRelationships(
	items []domain.ProfileRelationship,
	limit int,
	cursor int64,
) ([]domain.ProfileRelationship, int64, bool) {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if !hasMore {
		return items, 0, false
	}
	return items, cursor + int64(len(items)), true
}

func paginateSuggestions(
	items []domain.RelationshipSuggestion,
	limit int,
	cursor int64,
) ([]domain.RelationshipSuggestion, int64, bool) {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if !hasMore {
		return items, 0, false
	}
	return items, cursor + int64(len(items)), true
}

func normalizeRelationshipError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRelationshipNotFound
	}
	if isUniqueViolation(err) {
		return ErrRelationshipConflict
	}
	return err
}

func minID(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func maxID(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
