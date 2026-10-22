package repository

import (
	"errors"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func int64Pointer(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}

	result := value.Int64
	return &result
}

func textPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func genderPointer(value pgtype.Text) *domain.Gender {
	if !value.Valid {
		return nil
	}
	result := domain.Gender(value.String)
	return &result
}

func datePointer(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func timestampPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableGender(value *domain.Gender) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

type rowScanner interface {
	Scan(dest ...any) error
}

func normalizeLimit(limit int, limit_max int) int {
	if limit <= 0 {
		return 10
	}
	if limit > limit_max {
		return limit_max
	}
	return limit
}

func normalizeCursor(cursor int64) int64 {
	if cursor < 0 {
		return 0
	}
	return cursor
}

// isUniqueViolation проверяет, что база отклониа значение из-за UNIQUE-ограничения
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
