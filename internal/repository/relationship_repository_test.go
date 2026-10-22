package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/pashagolub/pgxmock/v4"
)

func relationshipRows(masterID, slaveID int64, status, typeName string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"profile_id_master", "profile_id_slave", "relationships_status", "relationships_type",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		masterID,
		slaveID,
		status,
		typeName,
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		nil,
		nil,
	)
}

func shortProfileRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "nickname", "profile_name", "surname", "patronymic", "avatar_url"})
	for _, id := range ids {
		rows.AddRow(id, "user", "User", "Example", nil, "/api/media/avatar.png")
	}
	return rows
}

func TestRelationshipRepositoryCreateListAndTypes(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewRelationshipRepository(mock)

	if _, err := repository.Create(ctx, 1, domain.CreateRelationshipRequest{ProfileIDSlave: 1, RelationshipsType: "friendship"}); !errors.Is(err, ErrInvalidRelationship) {
		t.Fatalf("Create(invalid) error = %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(queryPattern(relationshipPairLockQuery)).
		WithArgs("profile-relationship:1:2").
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM profile").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(queryPattern(relationshipPairQuery)).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{
			"profile_id_master", "profile_id_slave", "relationships_status", "relationships_type",
			"created_at", "updated_at", "deleted_at",
		}))
	mock.ExpectExec(queryPattern(archiveRelationshipPairQuery)).
		WithArgs(int64(1), int64(2)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectQuery(queryPattern(createRelationshipQuery)).
		WithArgs(int64(1), int64(2), "friendship").
		WillReturnRows(relationshipRows(1, 2, "request", "friendship"))
	mock.ExpectCommit()
	mock.ExpectQuery(queryPattern(shortProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(shortProfileRows(2))
	created, err := repository.Create(ctx, 1, domain.CreateRelationshipRequest{
		ProfileIDSlave: 2, RelationshipsType: "friendship",
	})
	if err != nil || created.ProfileIDMaster != 1 || created.Profile.Nickname != "user" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}

	if _, err := repository.List(ctx, 0, domain.ListRelationshipsRequest{}); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("List(invalid) error = %v", err)
	}
	mock.ExpectQuery(queryPattern(listRelationshipsQuery)).
		WithArgs(int64(1), "accepted", "friendship", int64(0), 2).
		WillReturnRows(relationshipRows(1, 2, "accepted", "friendship"))
	mock.ExpectQuery(queryPattern(shortProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(shortProfileRows(2))
	response, err := repository.List(ctx, 1, domain.ListRelationshipsRequest{
		RelationshipsStatus: domain.RelationshipAccept,
		RelationshipsType:   "friendship",
		Limit:               1,
	})
	if err != nil || len(response.Relationships) != 1 || response.Relationships[0].Profile.ID != 2 {
		t.Fatalf("List() = %+v, %v", response, err)
	}

	mock.ExpectQuery(queryPattern(listIncomingRequestsQuery)).
		WithArgs(int64(1), "family", int64(0), 11).
		WillReturnRows(pgxmock.NewRows([]string{
			"profile_id_master", "profile_id_slave", "relationships_status", "relationships_type",
			"created_at", "updated_at", "deleted_at",
		}))
	response, err = repository.Requests(ctx, 1, "family", 0, 10)
	if err != nil || len(response.Relationships) != 0 {
		t.Fatalf("Requests() = %+v, %v", response, err)
	}

	if _, err := repository.Types(ctx, 0); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Types(invalid) error = %v", err)
	}
	mock.ExpectQuery(queryPattern(listTypesQuery)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"relationships_type"}).AddRow("family").AddRow("friendship"))
	types, err := repository.Types(ctx, 1)
	if err != nil || len(types.RelationshipsTypes) != 2 {
		t.Fatalf("Types() = %+v, %v", types, err)
	}
}

func TestRelationshipRepositorySuggestionsAndActions(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewRelationshipRepository(mock)

	if _, err := repository.Suggestions(ctx, 0, 0, 10); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Suggestions(invalid) error = %v", err)
	}
	mock.ExpectQuery(queryPattern(myConnectionsQuery)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}).AddRow(2))
	mock.ExpectQuery(queryPattern(suggestionCandidatesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id", "common_count"}).AddRow(3, 2))
	mock.ExpectQuery(queryPattern(shortProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(shortProfileRows(3))
	mock.ExpectQuery(queryPattern(directRelationshipProfilesQuery)).
		WithArgs(int64(1), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}))
	suggestions, err := repository.Suggestions(ctx, 1, 0, 10)
	if err != nil || len(suggestions.Profiles) != 1 || suggestions.Profiles[0].CommonCount != 2 {
		t.Fatalf("Suggestions() = %+v, %v", suggestions, err)
	}

	mock.ExpectQuery(queryPattern(acceptRelationshipQuery)).
		WithArgs(int64(1), int64(2), "friendship").
		WillReturnRows(relationshipRows(1, 2, "accepted", "friendship"))
	mock.ExpectQuery(queryPattern(shortProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(shortProfileRows(1))
	accepted, err := repository.Accept(ctx, 2, domain.AcceptRelationshipRequest{ProfileID: 1, RelationshipsType: "friendship"})
	if err != nil || accepted.Profile.ID != 1 {
		t.Fatalf("Accept() = %+v, %v", accepted, err)
	}

	mock.ExpectQuery(queryPattern(cancelRelationshipQuery)).
		WithArgs(int64(1), int64(2), "friendship").
		WillReturnRows(relationshipRows(1, 2, "canceled", "friendship"))
	mock.ExpectQuery(queryPattern(shortProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(shortProfileRows(2))
	canceled, err := repository.Cancel(ctx, 1, domain.CancleRelationshipRequest{ProfileID: 2, RelationshipsType: "friendship"})
	if err != nil || canceled.Profile.ID != 2 {
		t.Fatalf("Cancel() = %+v, %v", canceled, err)
	}

	if _, err := repository.Accept(ctx, 0, domain.AcceptRelationshipRequest{}); !errors.Is(err, ErrInvalidRelationship) {
		t.Fatalf("Accept(invalid) error = %v", err)
	}
	if _, err := repository.Cancel(ctx, 0, domain.CancleRelationshipRequest{}); !errors.Is(err, ErrInvalidRelationship) {
		t.Fatalf("Cancel(invalid) error = %v", err)
	}
}

func TestRelationshipHelpers(t *testing.T) {
	items := []domain.ProfileRelationship{{ProfileIDMaster: 1, ProfileIDSlave: 2}}
	page, next, more := paginateRelationships(items, 1, 0)
	if len(page) != 1 || next != 0 || more {
		t.Fatal("relationship pagination is incorrect")
	}
	page, next, more = paginateRelationships(append(items, domain.ProfileRelationship{ProfileIDMaster: 1, ProfileIDSlave: 3}), 1, 0)
	if len(page) != 1 || next != 1 || !more {
		t.Fatal("relationship pagination with next page is incorrect")
	}
	suggestions := []domain.RelationshipSuggestion{{}, {}}
	if len(suggestionPage(suggestions, 1, 1)) != 1 || len(suggestionPage(suggestions, 3, 1)) != 0 {
		t.Fatal("suggestion page is incorrect")
	}
	if minID(2, 1) != 1 || maxID(1, 2) != 2 {
		t.Fatal("minID/maxID is incorrect")
	}
}
