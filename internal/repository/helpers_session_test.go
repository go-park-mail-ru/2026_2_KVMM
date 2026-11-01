package repository

import (
	"errors"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestHelpers(t *testing.T) {
	t.Parallel()

	value := int64(7)
	text := "value"
	gender := domain.GenderMale
	date := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	timestamp := date.Add(time.Hour)

	if got := int64Pointer(pgtype.Int8{Int64: value, Valid: true}); got == nil || *got != value {
		t.Fatalf("int64Pointer() = %v", got)
	}
	if int64Pointer(pgtype.Int8{}) != nil {
		t.Fatal("invalid int64 must return nil")
	}
	if got := textPointer(pgtype.Text{String: text, Valid: true}); got == nil || *got != text {
		t.Fatalf("textPointer() = %v", got)
	}
	if textPointer(pgtype.Text{}) != nil {
		t.Fatal("invalid text must return nil")
	}
	if got := genderPointer(pgtype.Text{String: string(gender), Valid: true}); got == nil || *got != gender {
		t.Fatalf("genderPointer() = %v", got)
	}
	if genderPointer(pgtype.Text{}) != nil {
		t.Fatal("invalid gender must return nil")
	}
	if got := datePointer(pgtype.Date{Time: date, Valid: true}); got == nil || !got.Equal(date) {
		t.Fatalf("datePointer() = %v", got)
	}
	if datePointer(pgtype.Date{}) != nil {
		t.Fatal("invalid date must return nil")
	}
	if got := timestampPointer(pgtype.Timestamptz{Time: timestamp, Valid: true}); got == nil || !got.Equal(timestamp) {
		t.Fatalf("timestampPointer() = %v", got)
	}
	if timestampPointer(pgtype.Timestamptz{}) != nil {
		t.Fatal("invalid timestamp must return nil")
	}

	if nullableInt64(&value) != value || nullableInt64(nil) != nil {
		t.Fatal("nullableInt64() returned an unexpected value")
	}
	if nullableString(&text) != text || nullableString(nil) != nil {
		t.Fatal("nullableString() returned an unexpected value")
	}
	empty := "   "
	if nullableString(&empty) != nil {
		t.Fatal("blank string must become nil")
	}
	if nullableGender(&gender) != string(gender) || nullableGender(nil) != nil {
		t.Fatal("nullableGender() returned an unexpected value")
	}
	if nullableTime(&timestamp) != timestamp || nullableTime(nil) != nil {
		t.Fatal("nullableTime() returned an unexpected value")
	}

	if normalizeLimit(0, 10) != 10 || normalizeLimit(50, 10) != 10 || normalizeLimit(3, 10) != 3 {
		t.Fatal("normalizeLimit() returned an unexpected value")
	}
	if normalizeCursor(-1) != 0 || normalizeCursor(5) != 5 {
		t.Fatal("normalizeCursor() returned an unexpected value")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23505"}) == false {
		t.Fatal("unique violation was not detected")
	}
	if isUniqueViolation(errors.New("other error")) {
		t.Fatal("ordinary error was detected as unique violation")
	}
}

func TestSessionRepository(t *testing.T) {
	t.Parallel()

	repository := NewSessionRepository()
	if _, _, err := repository.Read(""); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read(empty) error = %v", err)
	}
	token, csrf, err := repository.Create(42)
	if err != nil || token == "" || csrf == "" {
		t.Fatalf("Create() = %q, %q, %v", token, csrf, err)
	}
	userID, storedCSRF, err := repository.Read(token)
	if err != nil || userID != 42 || storedCSRF != csrf {
		t.Fatalf("Read() = %d, %q, %v", userID, storedCSRF, err)
	}
	if err := repository.ValidateCSRF(token, csrf); err != nil {
		t.Fatalf("ValidateCSRF() error = %v", err)
	}
	if err := repository.ValidateCSRF(token, "wrong"); !errors.Is(err, ErrInvalidCSRF) {
		t.Fatalf("ValidateCSRF(wrong) error = %v", err)
	}
	repository.Delete(token)
	if _, _, err := repository.Read(token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Read(deleted) error = %v", err)
	}
}

func TestNormalizeErrors(t *testing.T) {
	t.Parallel()
	if !errors.Is(normalizeMessageError(pgx.ErrNoRows), ErrMessageNotFound) {
		t.Fatal("message not found was not normalized")
	}
	if !errors.Is(normalizeProfileError(pgx.ErrNoRows), ErrProfileNotFound) {
		t.Fatal("profile not found was not normalized")
	}
	if !errors.Is(normalizePostError(pgx.ErrNoRows), ErrPostNotFound) {
		t.Fatal("post not found was not normalized")
	}
	if !errors.Is(normalizeRelationshipError(pgx.ErrNoRows), ErrRelationshipNotFound) {
		t.Fatal("relationship not found was not normalized")
	}
	if !errors.Is(normalizeAvatarError(pgx.ErrNoRows), ErrAvatarNotFound) {
		t.Fatal("avatar not found was not normalized")
	}
	if normalizeMessageError(nil) != nil || normalizeProfileError(nil) != nil || normalizePostError(nil) != nil {
		t.Fatal("nil error was changed")
	}
}
