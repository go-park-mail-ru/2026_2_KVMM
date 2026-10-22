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
	mock.ExpectExec(queryPattern(upsertParticipantQuery)).
		WithArgs(int64(1), int64(7)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(queryPattern(upsertParticipantQuery)).
		WithArgs(int64(2), int64(7)).
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
	mock.ExpectQuery(queryPattern(readParticipantsQuery)).
		WithArgs(int64(7)).
		WillReturnRows(profileRows(2, "bob"))
	mock.ExpectQuery(queryPattern(readLastMessageQuery)).
		WithArgs(int64(7)).
		WillReturnRows(messageRows(9))
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
