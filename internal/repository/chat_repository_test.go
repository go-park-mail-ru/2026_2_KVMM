package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func chatRows(ids ...int64) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "uid", "created_at"})
	for _, id := range ids {
		rows.AddRow(id, pgtype.UUID{Bytes: [16]byte{1}, Valid: true}, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	}
	return rows
}

func chatParticipantRows(chatID, profileID int64, nickname string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"chat_id", "id", "nickname", "email", "phone_number", "profile_name", "surname",
		"patronymic", "gender", "birthday", "bio", "city", "education", "workplace",
		"created_at", "updated_at", "deleted_at",
		"uid", "avatar_url",
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
		"Moscow",
		nil,
		nil,
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		nil,
		nil,
		pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		"/api/media/avatars/bob.png",
	)
}

func chatMessageRows(chatID, messageID int64) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "uid", "chat_id", "author_id", "message_text", "repost_from", "reply_from",
		"created_at", "updated_at", "deleted_at", "author_uid",
	}).AddRow(
		messageID,
		pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		chatID,
		2,
		"message",
		nil,
		nil,
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		nil,
		nil,
		pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
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
	if got := response.Chats[0].Participants[0].AvatarURLs; len(got) != 1 || got[0] != "/api/media/avatars/bob.png" {
		t.Fatalf("ListChats() avatar_urls = %v", got)
	}
	if city := response.Chats[0].Participants[0].City; city == nil || *city != "Moscow" {
		t.Fatalf("ListChats() city = %v", city)
	}
	if got := response.Chats[0].LastMessage.AuthorUID; got != "02000000-0000-0000-0000-000000000000" {
		t.Fatalf("ListChats() last message author_uid = %q", got)
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
