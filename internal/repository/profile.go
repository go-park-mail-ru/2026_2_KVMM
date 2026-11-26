package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileConflict = errors.New("profile unique value conflict")
)

const profileColumns = `
    id,
    nickname,
    email,
    phone_number,
    profile_name,
    surname,
    patronymic,
    gender,
    birthday,
    bio,
    city,
    education,
    workplace,
    created_at,
    updated_at,
    deleted_at,
    uid`

const updateProfileQuery = `
	UPDATE profile SET %s
	WHERE id = $%d
	AND deleted_at IS NULL
	RETURNING ` + profileColumns

const createProfileQuery = `
	INSERT INTO profile (
		nickname,
		email,
		password_hash
	)
	VALUES ($1, $2, $3)
	RETURNING ` + profileColumns

const readProfileByLoginQuery = `
	SELECT ` + profileColumns + `
	FROM profile
	WHERE deleted_at IS NULL
		AND (
			LOWER(nickname) = LOWER($1)
			OR LOWER(email) = LOWER($1)
			OR phone_number = $1
	)
	LIMIT 1`

const readByLoginAuthQuery = `
	SELECT ` + profileColumns + `, password_hash
	FROM profile
	WHERE deleted_at IS NULL
		AND (
			LOWER(nickname) = LOWER($1)
			OR LOWER(email) = LOWER($1)
			OR phone_number = $1
		)
	LIMIT 1`

const readProfileByIDQuery = `
	SELECT ` + profileColumns +
	` FROM profile WHERE id = $1 AND deleted_at IS NULL`

const searchProfileByUUIDQuery = `
	SELECT ` + profileColumns + `
	FROM profile
	WHERE uid = $1::uuid
		AND deleted_at IS NULL`

const readProfilesByIDsQuery = `
	SELECT ` + profileColumns + `,
		(
			SELECT m.media_path
			FROM profile_avatar pa
			JOIN media m ON m.id = pa.media_id
			WHERE pa.profile_id = p.id
				AND pa.deleted_at IS NULL
				AND m.deleted_at IS NULL
			ORDER BY pa.created_at DESC, pa.media_id DESC
			LIMIT 1
		) AS avatar_url
	FROM profile p
	WHERE p.id = ANY($1::BIGINT[])
		AND p.deleted_at IS NULL`

const listProfilesForSearchQuery = `
	SELECT ` + profileColumns + `
	FROM profile
	WHERE deleted_at IS NULL
	ORDER BY id
	LIMIT $1 OFFSET $2`

const deleteProfileQuery = `
	UPDATE profile SET deleted_at = NOW() 
	WHERE id = $1 AND deleted_at IS NULL`

// ProfileRepository выполняет sql-операции с таблицей profile
type ProfileRepository struct {
	pool dbPool
}

// NewProfileRepository создаёт репозиторий пользователей
func NewProfileRepository(pool dbPool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

// CreateProfileParams содержит данные для создания пользователя
type CreateProfileParams struct {
	Nickname string
	Email    *string

	PasswordHash string
}

// UpdateProfileParams содержит поля пользователя, которые нужно изменить
type UpdateProfileParams struct {
	Nickname    *string
	Email       *string
	PhoneNumber *string
	ProfileName *string
	Surname     *string
	Patronymic  *string
	Gender      *domain.Gender
	Birthday    *time.Time
	Bio         *string
	City        *string
	Education   *string
	Workplace   *string

	PasswordHash *string
}

// AuthProfile содержит профиль и данные, необходимые для проверки пароля
type AuthProfile struct {
	Profile      domain.Profile
	PasswordHash string `json:"-"`
}

// Create вставляет пользователя и возвращает созданный профиль
func (r *ProfileRepository) Create(
	ctx context.Context,
	params CreateProfileParams,
) (domain.Profile, error) {

	row := r.pool.QueryRow(
		ctx,
		createProfileQuery,

		strings.TrimSpace(params.Nickname),
		nullableString(params.Email),
		params.PasswordHash,
	)

	profile, _, err := scanProfile(row, false)
	return profile, normalizeProfileError(err)
}

// ReadByID возвращает активный профиль по ID
func (r *ProfileRepository) ReadByID(
	ctx context.Context,
	id int64,
) (domain.Profile, error) {
	profile, _, err := scanProfile(r.pool.QueryRow(ctx, readProfileByIDQuery, id), false)
	return profile, normalizeProfileError(err)
}

// SearchByUUID возвращает публичный профиль по UUID
func (r *ProfileRepository) SearchByUUID(
	ctx context.Context,
	uuid string,
) (domain.Profile, error) {
	profile, _, err := scanProfile(r.pool.QueryRow(
		ctx,
		searchProfileByUUIDQuery,
		strings.TrimSpace(uuid),
	), false)
	return profile, normalizeProfileError(err)
}

// ReadProfilesByIDs возвращает публичные данные нескольких профилей для внутренних сервисов.
func (r *ProfileRepository) ReadProfilesByIDs(ctx context.Context, ids []int64) ([]domain.Profile, error) {
	if len(ids) == 0 {
		return []domain.Profile{}, nil
	}
	rows, err := r.pool.Query(ctx, readProfilesByIDsQuery, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := make([]domain.Profile, 0, len(ids))
	for rows.Next() {
		var avatarURL pgtype.Text
		profile, _, scanErr := scanProfileWithLeadingAvatar(rows, nil, &avatarURL)
		if scanErr != nil {
			return nil, scanErr
		}
		if avatarURL.Valid {
			profile.AvatarURLs = []string{avatarURL.String}
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

// ListForSearch возвращает страницу активных профилей для пересборки поискового индекса
func (r *ProfileRepository) ListForSearch(ctx context.Context, offset, limit int) ([]domain.Profile, error) {
	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("invalid search page: offset=%d limit=%d", offset, limit)
	}
	rows, err := r.pool.Query(ctx, listProfilesForSearchQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]domain.Profile, 0)
	for rows.Next() {
		profile, _, err := scanProfile(rows, false)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profiles, nil
}

// ReadByLogin возвращает публичный профиль без password_hash и других данных авторизации
func (r *ProfileRepository) ReadByLogin(
	ctx context.Context,
	login string,
) (domain.Profile, error) {

	profile, _, err := scanProfile(r.pool.QueryRow(
		ctx,
		readProfileByLoginQuery,
		strings.TrimSpace(login)),
		false,
	)
	return profile, normalizeProfileError(err)
}

// ReadByLoginAuth ищет пользователя по nickname, email или номеру телефона для входа
func (r *ProfileRepository) ReadByLoginAuth(
	ctx context.Context,
	login string,
) (AuthProfile, error) {

	profile, passwordHash, err := scanProfile(
		r.pool.QueryRow(ctx, readByLoginAuthQuery, strings.TrimSpace(login)),
		true,
	)
	if err != nil {
		return AuthProfile{}, normalizeProfileError(err)
	}
	return AuthProfile{
		Profile:      profile,
		PasswordHash: passwordHash,
	}, nil
}

// Update изменяет переданные поля активного профиля
func (r *ProfileRepository) Update(
	ctx context.Context,
	id int64,
	params UpdateProfileParams,
) (domain.Profile, error) {
	sets := make([]string, 0, 11)
	args := make([]any, 0, 11)
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if params.Nickname != nil {
		add("nickname", strings.TrimSpace(*params.Nickname))
	}
	if params.Email != nil {
		add("email", nullableString(params.Email))
	}
	if params.PhoneNumber != nil {
		add("phone_number", nullableString(params.PhoneNumber))
	}
	if params.ProfileName != nil {
		add("profile_name", nullableString(params.ProfileName))
	}
	if params.Surname != nil {
		add("surname", nullableString(params.Surname))
	}
	if params.Patronymic != nil {
		add("patronymic", nullableString(params.Patronymic))
	}
	if params.Gender != nil {
		add("gender", nullableGender(params.Gender))
	}
	if params.Birthday != nil {
		add("birthday", nullableTime(params.Birthday))
	}
	if params.Bio != nil {
		add("bio", nullableString(params.Bio))
	}
	if params.City != nil {
		add("city", nullableString(params.City))
	}
	if params.Education != nil {
		add("education", nullableString(params.Education))
	}
	if params.Workplace != nil {
		add("workplace", nullableString(params.Workplace))
	}
	if params.PasswordHash != nil {
		add("password_hash", *params.PasswordHash)
	}
	if len(sets) == 0 {
		return r.ReadByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		updateProfileQuery,
		strings.Join(sets, ", "),
		len(args),
	)

	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, args...), false)
	return profile, normalizeProfileError(err)
}

// Delete помечает профиль удалённым soft delete
func (r *ProfileRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	command, err := r.pool.Exec(
		ctx,
		deleteProfileQuery,
		id,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// scanProfile сериализует ответ базы в domain.Profile
// withCredentials - добавлять ли приватные поля (хэш пароля)
func scanProfile(row rowScanner, withCredentials bool) (domain.Profile, string, error) {
	return scanProfileWithLeading(row, nil, withCredentials)
}

// scanProfileWithLeading сканирует профиль с дополнительным полем перед данными профиля
func scanProfileWithLeading(
	row rowScanner,
	leading any,
	withCredentials bool,
) (domain.Profile, string, error) {
	return scanProfileValues(row, leading, nil, withCredentials)
}

// scanProfileWithLeadingAvatar сканирует профиль и добавляет URL последней аватарки
func scanProfileWithLeadingAvatar(
	row rowScanner,
	leading any,
	avatarURL *pgtype.Text,
) (domain.Profile, string, error) {
	return scanProfileValues(row, leading, avatarURL, false)
}

func scanProfileValues(
	row rowScanner,
	leading any,
	avatarURL *pgtype.Text,
	withCredentials bool,
) (domain.Profile, string, error) {
	var (
		id           int64
		nickname     pgtype.Text
		email        pgtype.Text
		phoneNumber  pgtype.Text
		profileName  pgtype.Text
		surname      pgtype.Text
		patronymic   pgtype.Text
		gender       pgtype.Text
		bio          pgtype.Text
		city         pgtype.Text
		education    pgtype.Text
		workplace    pgtype.Text
		birthday     pgtype.Date
		createdAt    pgtype.Timestamptz
		updatedAt    pgtype.Timestamptz
		deletedAt    pgtype.Timestamptz
		uid          pgtype.UUID
		passwordHash pgtype.Text
	)

	dest := make([]any, 0, 15)
	if leading != nil {
		dest = append(dest, leading)
	}
	dest = append(dest,
		&id,
		&nickname,
		&email,
		&phoneNumber,
		&profileName,
		&surname,
		&patronymic,
		&gender,
		&birthday,
		&bio,
		&city,
		&education,
		&workplace,
		&createdAt,
		&updatedAt,
		&deletedAt,
		&uid,
	)
	if avatarURL != nil {
		dest = append(dest, avatarURL)
	}
	if withCredentials {
		dest = append(dest, &passwordHash)
	}

	if err := row.Scan(dest...); err != nil {
		return domain.Profile{}, "", err
	}

	profile := domain.Profile{
		ID:          id,
		Nickname:    nickname.String,
		Email:       textPointer(email),
		PhoneNumber: textPointer(phoneNumber),
		ProfileName: textPointer(profileName),
		Surname:     textPointer(surname),
		Patronymic:  textPointer(patronymic),
		Gender:      genderPointer(gender),
		Birthday:    datePointer(birthday),
		Bio:         textPointer(bio),
		City:        textPointer(city),
		Education:   textPointer(education),
		Workplace:   textPointer(workplace),
		CreatedAt:   createdAt.Time,
		UpdatedAt:   timestampPointer(updatedAt),
		DeletedAt:   timestampPointer(deletedAt),
		UID:         uuidString(uid),
	}

	var hash string
	if withCredentials {
		hash = passwordHash.String
	}

	return profile, hash, nil
}

func normalizeProfileError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProfileNotFound
	}
	if isUniqueViolation(err) {
		return ErrProfileConflict
	}
	return err
}
