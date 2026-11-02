package chat

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	chatmock "kvmm/internal/mocks/chat"
	sessionmock "kvmm/internal/mocks/session"
	chusecase "kvmm/internal/usecase/chat"
	sessionusecase "kvmm/internal/usecase/session"
)

// newChatHandler создаёт обработчик с gomock-зависимостями для тестов HTTP и WebSocket.
func newChatHandler(t *testing.T) *ChatHandler {
	t.Helper()
	ctrl := gomock.NewController(t)
	chats := chatmock.NewMockchatRepository(ctrl)
	messages := chatmock.NewMockmessageRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).AnyTimes()
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).AnyTimes()
	chats.EXPECT().IsParticipant(gomock.Any(), int64(1), int64(7)).Return(nil).AnyTimes()
	messages.EXPECT().ListMessages(gomock.Any(), int64(1), gomock.Any(), gomock.Any()).Return(
		domain.MessagesResponse{ChatID: 1}, nil,
	).AnyTimes()
	messages.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Message{ID: 1, ChatID: 1}, nil).AnyTimes()
	chats.EXPECT().Create(gomock.Any(), int64(7), int64(2)).Return(domain.Chat{ID: 1}, nil).AnyTimes()
	chats.EXPECT().ListChats(gomock.Any(), int64(7), gomock.Any(), gomock.Any()).Return(
		domain.ChatsResponse{Chats: []domain.Chat{{ID: 1}}}, nil,
	).AnyTimes()

	sessions := sessionusecase.New(session)
	return NewChatHandler(chusecase.New(chats, messages, sessions), sessions)
}

func chatRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	return request
}

func TestChatHandlers(t *testing.T) {
	h := newChatHandler(t)
	w := httptest.NewRecorder()
	h.Create(w, chatRequest(http.MethodPost, "/api/chats/create", `{"profile_id":2}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("Create() status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.List(w, chatRequest(http.MethodGet, "/api/chats/list", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("List() status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.Messages(w, chatRequest(http.MethodGet, "/api/chats/messages?chat_id=1", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("Messages() status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.Send(w, chatRequest(http.MethodPost, "/api/chats/send", `{"chat_id":1,"message_text":"hello"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("Send() status = %d", w.Code)
	}
}

func TestChatHandlerValidation(t *testing.T) {
	h := newChatHandler(t)
	w := httptest.NewRecorder()
	h.Messages(w, chatRequest(http.MethodGet, "/api/chats/messages?chat_id=bad", ""))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid chat id status = %d", w.Code)
	}

	w = httptest.NewRecorder()
	h.Send(w, chatRequest(http.MethodPost, "/api/chats/send", `{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid message status = %d", w.Code)
	}
}

func TestChatErrorMapping(t *testing.T) {
	h := newChatHandler(t)
	for _, test := range []struct {
		err    error
		status int
	}{
		{chusecase.ErrUnauthorized, http.StatusUnauthorized},
		{chusecase.ErrInvalidCSRF, http.StatusForbidden},
		{chusecase.ErrChatNotFound, http.StatusNotFound},
		{chusecase.ErrNotParticipant, http.StatusForbidden},
		{errors.New("internal"), http.StatusInternalServerError},
	} {
		w := httptest.NewRecorder()
		h.writeError(w, test.err)
		if w.Code != test.status {
			t.Errorf("writeError(%v) status = %d", test.err, w.Code)
		}
	}
}
