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

func chatRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "created_at"})
	for _, id := range ids {
		rows.AddRow(id, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	}
	return rows
}

func chatParticipantRows(chatID, profileID int64, nickname string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"chat_id", "id", "nickname", "email", "phone_number", "profile_name", "surname",
		"patronymic", "gender", "birthday", "bio", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		chatID,
		profileID,
		nickname,
		"user@example.com",
		nil,
		"User",
		"Example",
		nil,
		"male",
		nil,
		"bio",
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		nil,
		nil,
	)
}

func chatMessageRows(chatID, messageID int64) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "chat_id", "author_id", "message_text", "repost_from", "reply_from",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		messageID,
		chatID,
		2,
		"message",
		nil,
		nil,
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		nil,
		nil,
	)
}

func TestChatRepositoryCreate(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewChatRepository(mock)

	if _, err := repository.Create(ctx, 0, 2); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("Create(invalid) error = %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(queryPattern(acquirePrivateChatLockQuery)).
		WithArgs("private-chat:1:2").
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	mock.ExpectQuery(queryPattern(countActiveProfilesQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(queryPattern(findPrivateChatQuery)).
		WithArgs(int64(1), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"chat_id"}).AddRow(7))
	mock.ExpectExec(queryPattern(upsertParticipantsQuery)).
		WithArgs(pgxmock.AnyArg(), int64(7)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	mock.ExpectQuery(queryPattern(readChatQuery)).
		WithArgs(int64(7)).
		WillReturnRows(chatRows(7))
	mock.ExpectQuery(queryPattern(readParticipantsQuery)).
		WithArgs(int64(7)).
		WillReturnRows(profileRows(1, "alice"))
	chat, err := repository.Create(ctx, 2, 1)
	if err != nil || chat.ID != 7 || len(chat.Participants) != 1 {
		t.Fatalf("Create() = %+v, %v", chat, err)
	}
}

func TestChatRepositoryListAndParticipant(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewChatRepository(mock)

	if _, err := repository.ListChats(ctx, 0, 0, 10); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("ListChats(invalid) error = %v", err)
	}
	mock.ExpectQuery(queryPattern(listChatsQuery)).
		WithArgs(int64(1), int64(0), 2).
		WillReturnRows(chatRows(7, 6))
	mock.ExpectQuery(queryPattern(readParticipantsBatchQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(chatParticipantRows(7, 2, "bob"))
	mock.ExpectQuery(queryPattern(readLastMessagesBatchQuery)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(chatMessageRows(7, 9))
	response, err := repository.ListChats(ctx, 1, -1, 1)
	if err != nil || len(response.Chats) != 1 || response.NextCursor != 7 || !response.HasMore || response.Chats[0].LastMessage == nil {
		t.Fatalf("ListChats() = %+v, %v", response, err)
	}

	mock.ExpectQuery(queryPattern(requireParticipantQuery)).
		WithArgs(int64(7), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}).AddRow(2))
	if err := repository.IsParticipant(ctx, 7, 2); err != nil {
		t.Fatalf("IsParticipant() error = %v", err)
	}
	mock.ExpectQuery(queryPattern(requireParticipantQuery)).
		WithArgs(int64(7), int64(3)).
		WillReturnError(pgx.ErrNoRows)
	if err := repository.IsParticipant(ctx, 7, 3); !errors.Is(err, ErrNotChatParticipant) {
		t.Fatalf("IsParticipant(missing) error = %v", err)
	}
}

func TestChatRepositoryValidateMessageReferences(t *testing.T) {
	ctx := context.Background()
	mock := newMockPool(t)
	repository := NewChatRepository(mock)
	repostID := int64(11)
	replyID := int64(12)

	mock.ExpectQuery(queryPattern(requireParticipantQuery)).
		WithArgs(int64(7), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}).AddRow(2))
	mock.ExpectQuery(queryPattern(checkRepostAccessQuery)).
		WithArgs(repostID, int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(queryPattern(checkReplyChatQuery)).
		WithArgs(replyID, int64(7)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	if err := repository.ValidateMessageReferences(ctx, 2, 7, &repostID, &replyID); err != nil {
		t.Fatalf("ValidateMessageReferences() error = %v", err)
	}

	mock.ExpectQuery(queryPattern(requireParticipantQuery)).
		WithArgs(int64(7), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}).AddRow(2))
	mock.ExpectQuery(queryPattern(checkRepostAccessQuery)).
		WithArgs(repostID, int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	if err := repository.ValidateMessageReferences(ctx, 2, 7, &repostID, nil); !errors.Is(err, ErrRepostForbidden) {
		t.Fatalf("ValidateMessageReferences(repost) error = %v", err)
	}

	mock.ExpectQuery(queryPattern(requireParticipantQuery)).
		WithArgs(int64(7), int64(2)).
		WillReturnRows(pgxmock.NewRows([]string{"profile_id"}).AddRow(2))
	mock.ExpectQuery(queryPattern(checkReplyChatQuery)).
		WithArgs(replyID, int64(7)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	if err := repository.ValidateMessageReferences(ctx, 2, 7, nil, &replyID); !errors.Is(err, ErrReplyForbidden) {
		t.Fatalf("ValidateMessageReferences(reply) error = %v", err)
	}
}

func TestChatHelpers(t *testing.T) {
	page, next, more := paginateChats([]domain.Chat{{ID: 1}, {ID: 2}}, 1)
	if len(page) != 1 || next != 1 || !more {
		t.Fatal("chat pagination is incorrect")
	}
	page, next, more = paginateChats(nil, 1)
	if len(page) != 0 || next != 0 || more {
		t.Fatal("empty chat pagination is incorrect")
	}
}
