package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func postRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{
		"id", "uid", "post_text", "author_profile_id", "author_community_id", "created_at",
		"updated_at", "deleted_at",
	})
	for _, id := range ids {
		rows.AddRow(
			id,
			pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
			"post text",
			id,
			nil,
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			nil,
			nil,
		)
	}
	return rows
}

type postTestProfileService struct{}

func (postTestProfileService) GetProfileIDByUID(context.Context, string) (int64, error) {
	return 1, nil
}

func (postTestProfileService) GetProfilesByIDs(_ context.Context, ids []int64) ([]domain.Profile, error) {
	profiles := make([]domain.Profile, 0, len(ids))
	for _, id := range ids {
		name := "Alice"
		profiles = append(profiles, domain.Profile{ID: id, UID: "550e8400-e29b-41d4-a716-446655440000", Nickname: name, AvatarURLs: []string{"/api/media/avatar.png"}})
	}
	return profiles, nil
}

func TestPostRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewPostRepository(mock, postTestProfileService{})
	text := "hello"
	authorID := int64(1)

	if _, err := repository.Create(ctx, CreatePostParams{}); !errors.Is(err, ErrInvalidPostAuthor) {
		t.Fatalf("Create(without author) error = %v", err)
	}
	communityID := int64(2)
	if _, err := repository.Create(ctx, CreatePostParams{AuthorProfileID: &authorID, AuthorCommunityID: &communityID}); !errors.Is(err, ErrInvalidPostAuthor) {
		t.Fatalf("Create(with two authors) error = %v", err)
	}

	mock.ExpectQuery("(?s)INSERT INTO post .*RETURNING .*uid,").
		WithArgs(text, authorID, nil).
		WillReturnRows(postRows(1))
	created, err := repository.Create(ctx, CreatePostParams{PostText: &text, AuthorProfileID: &authorID})
	if err != nil || created.ID != 1 || created.Author.AvatarURL == "" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(postRows(1))
	if post, err := repository.ReadByID(ctx, 1); err != nil || post.ID != 1 {
		t.Fatalf("ReadByID() = %+v, %v", post, err)
	}

	newText := "updated"
	updateQuery := regexp.QuoteMeta("UPDATE post SET")
	mock.ExpectQuery(updateQuery).
		WithArgs(newText, int64(1)).
		WillReturnRows(postRows(1))
	if post, err := repository.Update(ctx, 1, UpdatePostParams{PostText: &newText}); err != nil || post.ID != 1 {
		t.Fatalf("Update() = %+v, %v", post, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(postRows(1))
	if _, err := repository.Update(ctx, 1, UpdatePostParams{}); err != nil {
		t.Fatalf("Update(empty) error = %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE post")).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := repository.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE post")).
		WithArgs(int64(2)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := repository.Delete(ctx, 2); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("Delete(missing) error = %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.id = $1")).
		WithArgs(int64(3)).
		WillReturnError(pgx.ErrNoRows)
	if _, err := repository.ReadByID(ctx, 3); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("ReadByID(missing) error = %v", err)
	}
}

func TestPostRepositoryLists(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewPostRepository(mock, postTestProfileService{})

	if _, _, _, err := repository.ReadProfilePosts(ctx, 0, 0, 10); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("ReadProfilePosts(invalid) error = %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.author_profile_id = $1")).
		WithArgs(int64(1), int64(0), 3).
		WillReturnRows(postRows(3, 2, 1))
	posts, next, more, err := repository.ReadProfilePosts(ctx, 1, -1, 2)
	if err != nil || len(posts) != 2 || next != 2 || !more {
		t.Fatalf("ReadProfilePosts() = %d, %d, %v, %v", len(posts), next, more, err)
	}

	if _, _, _, err := repository.ReadRelationshipPosts(ctx, 0, 0, 10, nil); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("ReadRelationshipPosts(invalid) error = %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("FROM profile_relationship pr")).
		WithArgs(int64(1), int64(0), 3, []string{"family"}).
		WillReturnRows(postRows(4))
	posts, next, more, err = repository.ReadRelationshipPosts(ctx, 1, 0, 2, []string{"family"})
	if err != nil || len(posts) != 1 || next != 4 || more {
		t.Fatalf("ReadRelationshipPosts() = %d, %d, %v, %v", len(posts), next, more, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY post.id DESC")).
		WithArgs(int64(0), 2).
		WillReturnRows(postRows(6, 5))
	posts, next, more, err = repository.ReadFeedPosts(ctx, -1, 1)
	if err != nil || len(posts) != 1 || next != 6 || !more {
		t.Fatalf("ReadFeedPosts() = %d, %d, %v, %v", len(posts), next, more, err)
	}
}

func TestPostPagination(t *testing.T) {
	posts, next, more := paginatePosts(nil, 2)
	if len(posts) != 0 || next != 0 || more {
		t.Fatal("empty post pagination is incorrect")
	}
	posts, next, more = paginatePosts([]domain.Post{{ID: 1}, {ID: 2}, {ID: 3}}, 2)
	if len(posts) != 2 || next != 2 || !more {
		t.Fatal("post pagination is incorrect")
	}
}
