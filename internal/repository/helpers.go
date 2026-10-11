package repository

import (
	"errors"
	"strings"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func normalizeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
