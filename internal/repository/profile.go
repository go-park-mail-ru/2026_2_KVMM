package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

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
func (r *ProfileRepository) Create(ctx context.Context, params CreateProfileParams) (domain.Profile, error) {
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
	return profile, normalizeError(err)
}

// ReadByID возвращает активный профиль по идентификатору
func (r *ProfileRepository) ReadByID(ctx context.Context, id int64) (domain.Profile, error) {
	query := `SELECT ` + profileColumns + ` FROM profile WHERE id = $1 AND deleted_at IS NULL`
	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, id), false)
	return profile, normalizeError(err)
}

// ReadByLogin возвращает публичный профиль без password_hash и других данных авторизации
func (r *ProfileRepository) ReadByLogin(ctx context.Context, login string) (domain.Profile, error) {
	query := `SELECT ` + profileColumns + `
        FROM profile
        WHERE deleted_at IS NULL
          AND (
            LOWER(nickname) = LOWER($1)
            OR LOWER(email) = LOWER($1)
            OR phone_number = $1
          )
        LIMIT 1`

	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, strings.TrimSpace(login)), false)
	return profile, normalizeError(err)
}

// ReadByLoginAuth ищет пользователя по nickname, email или номеру телефона для входа
func (r *ProfileRepository) ReadByLoginAuth(ctx context.Context, login string) (AuthProfile, error) {
	query := `SELECT ` + profileColumns + `, password_hash
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
		return AuthProfile{}, normalizeError(err)
	}
	return AuthProfile{
		Profile:      profile,
		PasswordHash: passwordHash,
	}, nil
}

// Update изменяет переданные поля активного профиля
func (r *ProfileRepository) Update(ctx context.Context, id int64, params UpdateProfileParams) (domain.Profile, error) {
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
	query := `UPDATE profile SET ` + strings.Join(sets, ", ") +
		fmt.Sprintf(` WHERE id = $%d AND deleted_at IS NULL RETURNING `, len(args)) + profileColumns

	profile, _, err := scanProfile(r.pool.QueryRow(ctx, query, args...), false)
	return profile, normalizeError(err)
}

// Delete помечает профиль удалённым soft delete
func (r *ProfileRepository) Delete(ctx context.Context, id int64) error {
	command, err := r.pool.Exec(
		ctx,
		`UPDATE profile SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// scanProfile сериализует ответ базы в domain.Profile
// withCredentials - добавлять ли приватные поля (хэш пароля)
func scanProfile(row rowScanner, withCredentials bool) (domain.Profile, string, error) {
	var (
		id                      int64
		nickname                string
		email, phoneNumber      pgtype.Text
		profileName, surname    pgtype.Text
		patronymic, gender, bio pgtype.Text
		birthday                pgtype.Date
		createdAt, updatedAt    pgtype.Timestamptz
		deletedAt               pgtype.Timestamptz
		passwordHash            string
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
		Nickname:    nickname,
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
	return profile, passwordHash, nil
}
