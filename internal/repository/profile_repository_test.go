package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet database expectations: %v", err)
		}
		mock.Close()
	})
	return mock
}

func queryPattern(query string) string {
	return regexp.QuoteMeta(query)
}

func profileRows(id int64, nickname string, passwordHash ...string) *pgxmock.Rows {
	createdAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	row := pgxmock.NewRows([]string{
		"id", "nickname", "email", "phone_number", "profile_name", "surname",
		"patronymic", "gender", "birthday", "bio", "created_at", "updated_at", "deleted_at",
	})
	values := []any{
		id, nickname, "user@example.com", nil, "User", "Example", nil, "male", nil,
		"bio", createdAt, nil, nil,
	}
	if len(passwordHash) > 0 {
		return pgxmock.NewRows([]string{
			"id", "nickname", "email", "phone_number", "profile_name", "surname",
			"patronymic", "gender", "birthday", "bio", "created_at", "updated_at", "deleted_at", "password_hash",
		}).AddRow(append(values, passwordHash[0])...)
	}
	return row.AddRow(values...)
}

func TestProfileRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewProfileRepository(mock)

	mock.ExpectQuery(queryPattern(createProfileQuery)).
		WithArgs("alice", "alice@example.com", "hash").
		WillReturnRows(profileRows(1, "alice"))
	created, err := repository.Create(ctx, CreateProfileParams{
		Nickname:     " alice ",
		Email:        stringPointer(" alice@example.com "),
		PasswordHash: "hash",
	})
	if err != nil || created.ID != 1 || created.Nickname != "alice" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}

	mock.ExpectQuery(queryPattern(readProfileByIDQuery)).
		WithArgs(int64(1)).
		WillReturnRows(profileRows(1, "alice"))
	if profile, err := repository.ReadByID(ctx, 1); err != nil || profile.ID != 1 {
		t.Fatalf("ReadByID() = %+v, %v", profile, err)
	}

	mock.ExpectQuery(queryPattern(readProfileByLoginQuery)).
		WithArgs("alice").
		WillReturnRows(profileRows(1, "alice"))
	if profile, err := repository.ReadByLogin(ctx, " alice "); err != nil || profile.Nickname != "alice" {
		t.Fatalf("ReadByLogin() = %+v, %v", profile, err)
	}

	mock.ExpectQuery(queryPattern(readByLoginAuthQuery)).
		WithArgs("alice").
		WillReturnRows(profileRows(1, "alice", "hash"))
	authProfile, err := repository.ReadByLoginAuth(ctx, " alice ")
	if err != nil || authProfile.PasswordHash != "hash" {
		t.Fatalf("ReadByLoginAuth() = %+v, %v", authProfile, err)
	}

	newNickname := "alice_new"
	newEmail := "new@example.com"
	updateQuery := fmt.Sprintf(
		updateProfileQuery,
		"nickname = $1, email = $2, updated_at = NOW()",
		3,
	)
	mock.ExpectQuery(queryPattern(updateQuery)).
		WithArgs(newNickname, newEmail, int64(1)).
		WillReturnRows(profileRows(1, "alice_new"))
	updated, err := repository.Update(ctx, 1, UpdateProfileParams{
		Nickname: &newNickname,
		Email:    &newEmail,
	})
	if err != nil || updated.Nickname != "alice_new" {
		t.Fatalf("Update() = %+v, %v", updated, err)
	}

	mock.ExpectQuery(queryPattern(readProfileByIDQuery)).
		WithArgs(int64(1)).
		WillReturnRows(profileRows(1, "alice_new"))
	if _, err := repository.Update(ctx, 1, UpdateProfileParams{}); err != nil {
		t.Fatalf("Update(empty) error = %v", err)
	}

	mock.ExpectExec(queryPattern(deleteProfileQuery)).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := repository.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	mock.ExpectExec(queryPattern(deleteProfileQuery)).
		WithArgs(int64(2)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := repository.Delete(ctx, 2); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Delete(missing) error = %v", err)
	}
}

func TestProfileRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewProfileRepository(mock)

	mock.ExpectQuery(queryPattern(readProfileByIDQuery)).
		WithArgs(int64(1)).
		WillReturnError(pgx.ErrNoRows)
	if _, err := repository.ReadByID(ctx, 1); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("ReadByID(missing) error = %v", err)
	}

	uniqueError := &pgconn.PgError{Code: "23505"}
	mock.ExpectQuery(queryPattern(createProfileQuery)).
		WithArgs("alice", nil, "hash").
		WillReturnError(uniqueError)
	if _, err := repository.Create(ctx, CreateProfileParams{Nickname: "alice", PasswordHash: "hash"}); !errors.Is(err, ErrProfileConflict) {
		t.Fatalf("Create(conflict) error = %v", err)
	}
}

func stringPointer(value string) *string { return &value }
