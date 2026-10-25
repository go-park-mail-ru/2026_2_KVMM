package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func messageRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{
		"id", "chat_id", "author_id", "message_text", "repost_from", "reply_from",
		"created_at", "updated_at", "deleted_at",
	})
	for _, id := range ids {
		var repost any
		var reply any
		if id == 12 {
			repost = int64(10)
			reply = int64(11)
		}
		rows.AddRow(
			id,
			1,
			2,
			"message",
			repost,
			reply,
			time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			nil,
			nil,
		)
	}
	return rows
}

func TestMessageRepositoryCreateAndList(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewMessageRepository(mock)

	text := "hello"
	repost := int64(10)
	reply := int64(11)
	mock.ExpectQuery(queryPattern(createMessageQuery)).
		WithArgs(int64(1), int64(2), text, repost, reply).
		WillReturnRows(messageRows(12))
	message, err := repository.Create(ctx, CreateMessageParams{
		ChatID: 1, AuthorID: 2, MessageText: &text, RepostFrom: &repost, ReplyFrom: &reply,
	})
	if err != nil || message.ID != 12 || message.RepostFrom == nil || *message.RepostFrom != 10 {
		t.Fatalf("Create() = %+v, %v", message, err)
	}

	if _, err := repository.ListMessages(ctx, 0, 0, 10); !errors.Is(err, ErrChatNotFound) {
		t.Fatalf("ListMessages(invalid) error = %v", err)
	}
	mock.ExpectQuery(queryPattern(listMessagesQuery)).
		WithArgs(int64(1), int64(0), 3).
		WillReturnRows(messageRows(12, 11, 10))
	mock.ExpectQuery(queryPattern(parentMessagesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(messageRows(10, 11))
	response, err := repository.ListMessages(ctx, 1, -1, 2)
	if err != nil || len(response.Messages) != 2 || !response.HasMore || response.NextCursor != 11 || len(response.Parents) != 2 {
		t.Fatalf("ListMessages() = %+v, %v", response, err)
	}

	mock.ExpectQuery(queryPattern(createMessageQuery)).
		WithArgs(int64(1), int64(2), nil, nil, nil).
		WillReturnError(pgx.ErrNoRows)
	if _, err := repository.Create(ctx, CreateMessageParams{ChatID: 1, AuthorID: 2}); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("Create(missing) error = %v", err)
	}
}

func TestMessageHelpers(t *testing.T) {
	messages := []domain.Message{
		{RepostFrom: int64PointerValue(10), ReplyFrom: int64PointerValue(11)},
		{RepostFrom: int64PointerValue(10)},
		{},
	}
	ids := collectParentIDs(messages)
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 11 {
		t.Fatalf("collectParentIDs() = %v", ids)
	}
	page, next, more := paginateMessages([]domain.Message{{ID: 1}, {ID: 2}, {ID: 3}}, 2)
	if len(page) != 2 || next != 2 || !more {
		t.Fatal("message pagination is incorrect")
	}
	page, next, more = paginateMessages(nil, 2)
	if len(page) != 0 || next != 0 || more {
		t.Fatal("empty message pagination is incorrect")
	}
}

func int64PointerValue(value int64) *int64 { return &value }
