package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func postRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{
		"id", "post_text", "author_profile_id", "author_community_id", "created_at",
		"updated_at", "deleted_at",
	})
	for _, id := range ids {
		rows.AddRow(
			id,
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

func postAuthorRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "author_name"})
	for _, id := range ids {
		rows.AddRow(id, "Alice")
	}
	return rows
}

func postAvatarRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"profile_id", "media_path"})
	for _, id := range ids {
		rows.AddRow(id, "/api/media/avatar.png")
	}
	return rows
}

func TestPostRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewPostRepository(mock)
	text := "hello"
	authorID := int64(1)

	if _, err := repository.Create(ctx, CreatePostParams{}); !errors.Is(err, ErrInvalidPostAuthor) {
		t.Fatalf("Create(without author) error = %v", err)
	}
	communityID := int64(2)
	if _, err := repository.Create(ctx, CreatePostParams{AuthorProfileID: &authorID, AuthorCommunityID: &communityID}); !errors.Is(err, ErrInvalidPostAuthor) {
		t.Fatalf("Create(with two authors) error = %v", err)
	}

	mock.ExpectQuery(queryPattern("INSERT INTO post (\n\t\t\tpost_text,\n\t\t\tauthor_profile_id,\n\t\t\tauthor_community_id\n\t\t)\n\t\tVALUES ($1, $2, $3)\n\t\tRETURNING \n\tid,\n\tpost_text,\n\tauthor_profile_id,\n\tauthor_community_id,\n\tcreated_at,\n\tupdated_at,\n\tdeleted_at")).
		WithArgs(text, authorID, nil).
		WillReturnRows(postRows(1))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(1))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(1))
	created, err := repository.Create(ctx, CreatePostParams{PostText: &text, AuthorProfileID: &authorID})
	if err != nil || created.ID != 1 || created.Author.AvatarURL == "" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(postRows(1))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(1))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(1))
	if post, err := repository.ReadByID(ctx, 1); err != nil || post.ID != 1 {
		t.Fatalf("ReadByID() = %+v, %v", post, err)
	}

	newText := "updated"
	updateQuery := regexp.QuoteMeta("UPDATE post SET")
	mock.ExpectQuery(updateQuery).
		WithArgs(newText, int64(1)).
		WillReturnRows(postRows(1))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(1))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(1))
	if post, err := repository.Update(ctx, 1, UpdatePostParams{PostText: &newText}); err != nil || post.ID != 1 {
		t.Fatalf("Update() = %+v, %v", post, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.id = $1")).
		WithArgs(int64(1)).
		WillReturnRows(postRows(1))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(1))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(1))
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
	repository := NewPostRepository(mock)

	if _, _, _, err := repository.ReadProfilePosts(ctx, 0, 0, 10); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("ReadProfilePosts(invalid) error = %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("WHERE post.author_profile_id = $1")).
		WithArgs(int64(1), int64(0), 3).
		WillReturnRows(postRows(3, 2, 1))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(3, 2))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(3, 2))
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
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(4))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(4))
	posts, next, more, err = repository.ReadRelationshipPosts(ctx, 1, 0, 2, []string{"family"})
	if err != nil || len(posts) != 1 || next != 4 || more {
		t.Fatalf("ReadRelationshipPosts() = %d, %d, %v, %v", len(posts), next, more, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY post.id DESC")).
		WithArgs(int64(0), 2).
		WillReturnRows(postRows(6, 5))
	mock.ExpectQuery(queryPattern(postAuthorsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAuthorRows(6))
	mock.ExpectQuery(queryPattern(postAuthorAvatarsQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(postAvatarRows(6))
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
