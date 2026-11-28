package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrOnlineConfEntryNotFound = errors.New("onlineconf entry not found")
	ErrOnlineConfEntryConflict = errors.New("onlineconf entry already exists")
	ErrOnlineConfAdminNotFound = errors.New("onlineconf admin not found")
	ErrOnlineConfAdminConflict = errors.New("onlineconf admin already exists")
)

const onlineConfEntryColumns = `
	id,
	scope,
	name,
	kind,
	value,
	rollout,
	created_by_profile_id,
	updated_by_profile_id,
	created_at,
	updated_at`

const onlineConfAdminColumns = `
	id,
	profile_id,
	profile_uid,
	granted_by_profile_id,
	created_at,
	updated_at,
	revoked_at`

const createOnlineConfEntryQuery = `
	INSERT INTO onlineconf_entry (
		scope,
		name,
		kind,
		value,
		rollout,
		created_by_profile_id,
		updated_by_profile_id
	)
	VALUES ($1, $2, $3, $4::jsonb, $5, $6, $6)
	RETURNING ` + onlineConfEntryColumns

const updateOnlineConfEntryQuery = `
	UPDATE onlineconf_entry
	SET value = COALESCE($2::jsonb, value),
		rollout = COALESCE($3, rollout),
		updated_by_profile_id = $4,
		updated_at = CURRENT_TIMESTAMP
	WHERE id = $1
	RETURNING ` + onlineConfEntryColumns

const readOnlineConfEntryQuery = `
	SELECT ` + onlineConfEntryColumns + `
	FROM onlineconf_entry
	WHERE id = $1`

const listOnlineConfEntriesQuery = `
	SELECT ` + onlineConfEntryColumns + `
	FROM onlineconf_entry
	WHERE (NULLIF($1, '') IS NULL OR scope = $1)
		AND (NULLIF($2, '') IS NULL OR name ILIKE '%' || $2 || '%')
	ORDER BY scope, name, id`

const listOnlineConfEntriesByScopeQuery = `
	SELECT ` + onlineConfEntryColumns + `
	FROM onlineconf_entry
	WHERE scope = $1
	ORDER BY name, id`

const listOnlineConfTargetsQuery = `
	SELECT entry_id, profile_id, profile_uid, mode
	FROM onlineconf_flag_target
	WHERE entry_id = ANY($1::BIGINT[])
	ORDER BY entry_id, profile_id`

const deleteOnlineConfTargetsQuery = `
	DELETE FROM onlineconf_flag_target
	WHERE entry_id = $1`

const insertOnlineConfTargetsQuery = `
	INSERT INTO onlineconf_flag_target (entry_id, profile_id, profile_uid, mode)
	SELECT $1, target.profile_id, target.profile_uid, target.mode
	FROM UNNEST($2::BIGINT[], $3::UUID[], $4::TEXT[])
		AS target(profile_id, profile_uid, mode)`

const createOnlineConfAdminQuery = `
	INSERT INTO onlineconf_admin (profile_id, profile_uid, granted_by_profile_id)
	VALUES ($1, $2::uuid, $3)
	RETURNING ` + onlineConfAdminColumns

const readOnlineConfAdminByProfileIDQuery = `
	SELECT ` + onlineConfAdminColumns + `
	FROM onlineconf_admin
	WHERE profile_id = $1
		AND revoked_at IS NULL`

// OnlineConfRepository выполняет SQL-операции сервиса OnlineConf
type OnlineConfRepository struct {
	pool dbPool
}

// CreateOnlineConfEntryParams содержит данные для создания конфигурации или feature flag
type CreateOnlineConfEntryParams struct {
	Scope              domain.OnlineConfScope
	Name               string
	Kind               domain.OnlineConfEntryKind
	Value              json.RawMessage
	Rollout            *int
	Targets            []domain.OnlineConfFlagTarget
	CreatedByProfileID int64
}

// UpdateOnlineConfEntryParams содержит изменяемые данные конфигурации или feature flag
type UpdateOnlineConfEntryParams struct {
	Value              *json.RawMessage
	Rollout            *int
	ReplaceTargets     bool
	Targets            []domain.OnlineConfFlagTarget
	UpdatedByProfileID int64
}

// CreateOnlineConfAdminParams содержит профиль Nexly, которому выдаётся доступ к админке
type CreateOnlineConfAdminParams struct {
	ProfileID          int64
	ProfileUID         string
	GrantedByProfileID *int64
}

type onlineConfQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// NewOnlineConfRepository создаёт репозиторий конфигураций и администраторов OnlineConf
func NewOnlineConfRepository(pool dbPool) *OnlineConfRepository {
	return &OnlineConfRepository{pool: pool}
}

// CreateEntry создаёт конфигурацию или feature flag вместе с персональными правилами в одной транзакции
func (r *OnlineConfRepository) CreateEntry(
	ctx context.Context,
	params CreateOnlineConfEntryParams,
) (domain.OnlineConfEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	entry, err := scanOnlineConfEntry(tx.QueryRow(
		ctx,
		createOnlineConfEntryQuery,
		string(params.Scope),
		strings.TrimSpace(params.Name),
		string(params.Kind),
		onlineConfJSONArgument(params.Value),
		onlineConfIntArgument(params.Rollout),
		params.CreatedByProfileID,
	))
	if err != nil {
		return domain.OnlineConfEntry{}, normalizeOnlineConfEntryError(err)
	}
	if err := insertOnlineConfTargets(ctx, tx, entry.ID, params.Targets); err != nil {
		return domain.OnlineConfEntry{}, normalizeOnlineConfEntryError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.OnlineConfEntry{}, err
	}
	committed = true
	applyOnlineConfTargets(&entry, params.Targets)
	return entry, nil
}

// UpdateEntry изменяет значение или раскатку записи и при необходимости полностью заменяет правила пользователей
func (r *OnlineConfRepository) UpdateEntry(
	ctx context.Context,
	id int64,
	params UpdateOnlineConfEntryParams,
) (domain.OnlineConfEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	entry, err := scanOnlineConfEntry(tx.QueryRow(
		ctx,
		updateOnlineConfEntryQuery,
		id,
		onlineConfJSONPointerArgument(params.Value),
		onlineConfIntArgument(params.Rollout),
		params.UpdatedByProfileID,
	))
	if err != nil {
		return domain.OnlineConfEntry{}, normalizeOnlineConfEntryError(err)
	}
	if params.ReplaceTargets {
		if _, err := tx.Exec(ctx, deleteOnlineConfTargetsQuery, entry.ID); err != nil {
			return domain.OnlineConfEntry{}, err
		}
		if err := insertOnlineConfTargets(ctx, tx, entry.ID, params.Targets); err != nil {
			return domain.OnlineConfEntry{}, normalizeOnlineConfEntryError(err)
		}
	}
	targetsByEntry, err := readOnlineConfTargets(ctx, tx, []int64{entry.ID})
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.OnlineConfEntry{}, err
	}
	committed = true
	applyOnlineConfTargets(&entry, targetsByEntry[entry.ID])
	return entry, nil
}

// ReadEntry возвращает запись OnlineConf вместе с персональными правилами feature flag.
func (r *OnlineConfRepository) ReadEntry(ctx context.Context, id int64) (domain.OnlineConfEntry, error) {
	entry, err := scanOnlineConfEntry(r.pool.QueryRow(ctx, readOnlineConfEntryQuery, id))
	if err != nil {
		return domain.OnlineConfEntry{}, normalizeOnlineConfEntryError(err)
	}
	targetsByEntry, err := readOnlineConfTargets(ctx, r.pool, []int64{entry.ID})
	if err != nil {
		return domain.OnlineConfEntry{}, err
	}
	applyOnlineConfTargets(&entry, targetsByEntry[entry.ID])
	return entry, nil
}

// ListEntries возвращает записи с необязательной фильтрацией по scope и подстроке имени.
func (r *OnlineConfRepository) ListEntries(
	ctx context.Context,
	scope domain.OnlineConfScope,
	query string,
) ([]domain.OnlineConfEntry, error) {
	rows, err := r.pool.Query(ctx, listOnlineConfEntriesQuery, string(scope), strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries, err := scanOnlineConfEntries(rows)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []domain.OnlineConfEntry{}, nil
	}
	targetsByEntry, err := readOnlineConfTargets(ctx, r.pool, onlineConfEntryIDs(entries))
	if err != nil {
		return nil, err
	}
	for index := range entries {
		applyOnlineConfTargets(&entries[index], targetsByEntry[entries[index].ID])
	}
	return entries, nil
}

// Snapshot возвращает данные одного scope
func (r *OnlineConfRepository) Snapshot(
	ctx context.Context,
	scope domain.OnlineConfScope,
) (domain.OnlineConfSnapshot, error) {
	rows, err := r.pool.Query(ctx, listOnlineConfEntriesByScopeQuery, string(scope))
	if err != nil {
		return domain.OnlineConfSnapshot{}, err
	}
	defer rows.Close()

	entries, err := scanOnlineConfEntries(rows)
	if err != nil {
		return domain.OnlineConfSnapshot{}, err
	}
	targetsByEntry, err := readOnlineConfTargets(ctx, r.pool, onlineConfEntryIDs(entries))
	if err != nil {
		return domain.OnlineConfSnapshot{}, err
	}

	snapshot := domain.OnlineConfSnapshot{
		Scope:   scope,
		Configs: make(map[string]json.RawMessage),
		Flags:   make(map[string]domain.OnlineConfFlagSnapshot),
	}
	for _, entry := range entries {
		switch entry.Kind {
		case domain.OnlineConfEntryConfig:
			snapshot.Configs[entry.Name] = entry.Value
		case domain.OnlineConfEntryFlag:
			flag := domain.OnlineConfFlagSnapshot{}
			if entry.Rollout != nil {
				flag.Rollout = *entry.Rollout
			}
			for _, target := range targetsByEntry[entry.ID] {
				if target.Mode == domain.OnlineConfTargetInclude {
					flag.IncludeIDs = append(flag.IncludeIDs, target.ProfileID)
				} else {
					flag.ExcludeIDs = append(flag.ExcludeIDs, target.ProfileID)
				}
			}
			snapshot.Flags[entry.Name] = flag
		}
	}
	return snapshot, nil
}

// CreateAdmin добавляет профиль в список администраторов OnlineConf
func (r *OnlineConfRepository) CreateAdmin(
	ctx context.Context,
	params CreateOnlineConfAdminParams,
) (domain.OnlineConfAdmin, error) {
	admin, err := scanOnlineConfAdmin(r.pool.QueryRow(
		ctx,
		createOnlineConfAdminQuery,
		params.ProfileID,
		strings.TrimSpace(params.ProfileUID),
		nullableInt64(params.GrantedByProfileID),
	))
	if err != nil {
		return domain.OnlineConfAdmin{}, normalizeOnlineConfAdminError(err)
	}
	return admin, nil
}

// ReadAdminByProfileID возвращает активное право администратора по ID
func (r *OnlineConfRepository) ReadAdminByProfileID(ctx context.Context, profileID int64) (domain.OnlineConfAdmin, error) {
	admin, err := scanOnlineConfAdmin(r.pool.QueryRow(ctx, readOnlineConfAdminByProfileIDQuery, profileID))
	if err != nil {
		return domain.OnlineConfAdmin{}, normalizeOnlineConfAdminError(err)
	}
	return admin, nil
}

func insertOnlineConfTargets(
	ctx context.Context,
	tx pgx.Tx,
	entryID int64,
	targets []domain.OnlineConfFlagTarget,
) error {
	if len(targets) == 0 {
		return nil
	}
	profileIDs := make([]int64, 0, len(targets))
	profileUIDs := make([]string, 0, len(targets))
	modes := make([]string, 0, len(targets))
	for _, target := range targets {
		profileIDs = append(profileIDs, target.ProfileID)
		profileUIDs = append(profileUIDs, target.ProfileUID)
		modes = append(modes, string(target.Mode))
	}
	_, err := tx.Exec(ctx, insertOnlineConfTargetsQuery, entryID, profileIDs, profileUIDs, modes)
	return err
}

func readOnlineConfTargets(
	ctx context.Context,
	queryer onlineConfQuerier,
	entryIDs []int64,
) (map[int64][]domain.OnlineConfFlagTarget, error) {
	targetsByEntry := make(map[int64][]domain.OnlineConfFlagTarget)
	if len(entryIDs) == 0 {
		return targetsByEntry, nil
	}
	rows, err := queryer.Query(ctx, listOnlineConfTargetsQuery, entryIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var entryID, profileID int64
		var profileUID pgtype.UUID
		var mode pgtype.Text
		if err := rows.Scan(&entryID, &profileID, &profileUID, &mode); err != nil {
			return nil, err
		}
		targetsByEntry[entryID] = append(targetsByEntry[entryID], domain.OnlineConfFlagTarget{
			ProfileID:  profileID,
			ProfileUID: uuidString(profileUID),
			Mode:       domain.OnlineConfFlagTargetMode(mode.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targetsByEntry, nil
}

func scanOnlineConfEntry(row rowScanner) (domain.OnlineConfEntry, error) {
	var (
		entry     domain.OnlineConfEntry
		scope     pgtype.Text
		kind      pgtype.Text
		value     []byte
		rollout   pgtype.Int2
		updatedAt pgtype.Timestamptz
	)
	if err := row.Scan(
		&entry.ID,
		&scope,
		&entry.Name,
		&kind,
		&value,
		&rollout,
		&entry.CreatedByProfileID,
		&entry.UpdatedByProfileID,
		&entry.CreatedAt,
		&updatedAt,
	); err != nil {
		return domain.OnlineConfEntry{}, err
	}
	entry.Scope = domain.OnlineConfScope(scope.String)
	entry.Kind = domain.OnlineConfEntryKind(kind.String)
	if value != nil {
		entry.Value = append(json.RawMessage(nil), value...)
	}
	if rollout.Valid {
		value := int(rollout.Int16)
		entry.Rollout = &value
	}
	entry.UpdatedAt = timestampPointer(updatedAt)
	return entry, nil
}

func scanOnlineConfEntries(rows pgx.Rows) ([]domain.OnlineConfEntry, error) {
	entries := make([]domain.OnlineConfEntry, 0)
	for rows.Next() {
		entry, err := scanOnlineConfEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func scanOnlineConfAdmin(row rowScanner) (domain.OnlineConfAdmin, error) {
	var (
		admin       domain.OnlineConfAdmin
		profileUID  pgtype.UUID
		grantedByID pgtype.Int8
		updatedAt   pgtype.Timestamptz
		revokedAt   pgtype.Timestamptz
	)
	if err := row.Scan(
		&admin.ID,
		&admin.ProfileID,
		&profileUID,
		&grantedByID,
		&admin.CreatedAt,
		&updatedAt,
		&revokedAt,
	); err != nil {
		return domain.OnlineConfAdmin{}, err
	}
	admin.ProfileUID = uuidString(profileUID)
	admin.GrantedByProfileID = int64Pointer(grantedByID)
	admin.UpdatedAt = timestampPointer(updatedAt)
	admin.RevokedAt = timestampPointer(revokedAt)
	return admin, nil
}

func applyOnlineConfTargets(entry *domain.OnlineConfEntry, targets []domain.OnlineConfFlagTarget) {
	entry.Include = nil
	entry.Exclude = nil
	for _, target := range targets {
		if target.Mode == domain.OnlineConfTargetInclude {
			entry.Include = append(entry.Include, target.ProfileUID)
		} else {
			entry.Exclude = append(entry.Exclude, target.ProfileUID)
		}
	}
}

func onlineConfEntryIDs(entries []domain.OnlineConfEntry) []int64 {
	ids := make([]int64, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func onlineConfJSONArgument(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func onlineConfJSONPointerArgument(value *json.RawMessage) any {
	if value == nil {
		return nil
	}
	return onlineConfJSONArgument(*value)
}

func onlineConfIntArgument(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func normalizeOnlineConfEntryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOnlineConfEntryNotFound
	}
	if isUniqueViolation(err) {
		return ErrOnlineConfEntryConflict
	}
	return err
}

func normalizeOnlineConfAdminError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOnlineConfAdminNotFound
	}
	if isUniqueViolation(err) {
		return ErrOnlineConfAdminConflict
	}
	return err
}
