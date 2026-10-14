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
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

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
    created_at,
    updated_at,
    deleted_at`

// ProfileRepository выполняет sql-операции с таблицей profile
type ProfileRepository struct {
	pool *pgxpool.Pool
}

// NewProfileRepository создаёт репозиторий пользователей
func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
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
	const query = `
        INSERT INTO profile (
            nickname,
            email,
            password_hash
        )
        VALUES ($1, $2, $3)
        RETURNING ` + profileColumns

	row := r.pool.QueryRow(
		ctx,
		query,

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
	const query = `SELECT ` + profileColumns + ` FROM profile WHERE id = $1 AND deleted_at IS NULL`
	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, id), false)
	return profile, normalizeProfileError(err)
}

// ReadByLogin возвращает публичный профиль без password_hash и других данных авторизации
func (r *ProfileRepository) ReadByLogin(
	ctx context.Context,
	login string,
) (domain.Profile, error) {
	const query = `SELECT ` + profileColumns + `
        FROM profile
        WHERE deleted_at IS NULL
        	AND (
        		LOWER(nickname) = LOWER($1)
        		OR LOWER(email) = LOWER($1)
        		OR phone_number = $1
        )
        LIMIT 1`

	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, strings.TrimSpace(login)), false)
	return profile, normalizeProfileError(err)
}

// ReadByLoginAuth ищет пользователя по nickname, email или номеру телефона для входа
func (r *ProfileRepository) ReadByLoginAuth(
	ctx context.Context,
	login string,
) (AuthProfile, error) {
	const query = `SELECT ` + profileColumns + `, password_hash
        FROM profile
        WHERE deleted_at IS NULL
        	AND (
            	LOWER(nickname) = LOWER($1)
				OR LOWER(email) = LOWER($1)
				OR phone_number = $1
        	)
        LIMIT 1`

	profile, passwordHash, err := scanProfile(
		r.pool.QueryRow(ctx, query, strings.TrimSpace(login)),
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
	if params.PasswordHash != nil {
		add("password_hash", *params.PasswordHash)
	}
	if len(sets) == 0 {
		return r.ReadByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	const updateProfileQuery = `UPDATE profile SET %s
		WHERE id = $%d
		AND deleted_at IS NULL
		RETURNING ` + postColumns
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
	const query = `UPDATE profile SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	command, err := r.pool.Exec(
		ctx,
		query,
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
		birthday     pgtype.Date
		createdAt    pgtype.Timestamptz
		updatedAt    pgtype.Timestamptz
		deletedAt    pgtype.Timestamptz
		passwordHash pgtype.Text
	)

	dest := []any{
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
		&createdAt,
		&updatedAt,
		&deletedAt,
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
		CreatedAt:   createdAt.Time,
		UpdatedAt:   timestampPointer(updatedAt),
		DeletedAt:   timestampPointer(deletedAt),
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
	return err
}
