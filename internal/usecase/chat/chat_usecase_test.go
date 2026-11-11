package chat

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	chatmock "kvmm/internal/mocks/chat"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

func TestChatUseCaseMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	chats := chatmock.NewMockchatRepository(ctrl)
	messages := chatmock.NewMockmessageRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(chats, messages, sessionusecase.New(session))
	ctx := context.Background()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).Times(4)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(4)
	chats.EXPECT().Create(ctx, int64(7), int64(2)).Return(domain.Chat{ID: 1}, nil)
	chats.EXPECT().ListChats(ctx, int64(7), int64(0), 10).Return(domain.ChatsResponse{Chats: []domain.Chat{{ID: 1}}}, nil)
	chats.EXPECT().IsParticipant(ctx, int64(1), int64(7)).Return(nil)
	messages.EXPECT().ListMessages(ctx, int64(1), int64(0), 10).Return(domain.MessagesResponse{ChatID: 1}, nil)
	text := "hello"
	messages.EXPECT().Create(ctx, gomock.Any()).Return(domain.Message{ID: 1}, nil)

	if _, err := useCase.Create(ctx, "session", "csrf", domain.CreateChatRequest{ProfileID: 2}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := useCase.List(ctx, "session", "csrf", 0, 10); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := useCase.Messages(ctx, "session", "csrf", 1, 0, 10); err != nil {
		t.Fatalf("Messages() error = %v", err)
	}
	if _, err := useCase.Send(ctx, "session", "csrf", domain.SendMessageRequest{ChatID: 1, MessageText: &text}); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
}

func TestChatUseCaseErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	chats := chatmock.NewMockchatRepository(ctrl)
	messages := chatmock.NewMockmessageRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(chats, messages, sessionusecase.New(session))
	ctx := context.Background()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	chats.EXPECT().ListChats(ctx, int64(7), int64(0), 10).Return(domain.ChatsResponse{}, errors.New("database unavailable"))
	if _, err := useCase.List(ctx, "session", "csrf", 0, 10); err == nil {
		t.Fatal("List() expected error")
	}

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	chats.EXPECT().IsParticipant(ctx, int64(1), int64(7)).Return(repository.ErrNotChatParticipant)
	if _, err := useCase.Messages(ctx, "session", "csrf", 1, 0, 10); !errors.Is(err, repository.ErrNotChatParticipant) {
		t.Fatalf("Messages() error = %v", err)
	}

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	if _, err := useCase.Send(ctx, "session", "csrf", domain.SendMessageRequest{ChatID: 1}); !errors.Is(err, repository.ErrEmptyMessage) {
		t.Fatalf("Send(empty) error = %v", err)
	}
}
