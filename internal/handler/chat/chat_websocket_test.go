package chat

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kvmm/internal/domain"
	chusecase "kvmm/internal/usecase/chat"

	"github.com/gorilla/websocket"
)

func TestWebSocketAuthenticationSubscriptionAndMessage(t *testing.T) {
	h := newChatHandler(t)
	server := httptest.NewServer(http.HandlerFunc(h.WebSocket))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Cookie": []string{"session_id=session"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))

	if err := connection.WriteJSON(domain.ChatWebSocketAuthRequest{Type: "auth", CSRFToken: "csrf"}); err != nil {
		t.Fatal(err)
	}
	var event domain.ChatWebSocketEvent
	if err := connection.ReadJSON(&event); err != nil || event.Type != "connected" {
		t.Fatalf("connected event = %+v, %v", event, err)
	}
	if err := connection.ReadJSON(&event); err != nil || event.Type != "subscribed" || event.ChatID != 1 {
		t.Fatalf("subscribed event = %+v, %v", event, err)
	}

	text := "hello"
	if err := connection.WriteJSON(domain.ChatWebSocketRequest{
		Type:    "message",
		ChatID:  1,
		Message: &domain.Message{MessageText: &text},
	}); err != nil {
		t.Fatal(err)
	}
	if err := connection.ReadJSON(&event); err != nil || event.Type != "message" || event.Message == nil {
		t.Fatalf("message event = %+v, %v", event, err)
	}
}

func TestWebSocketHelpers(t *testing.T) {
	if websocketError(chusecase.ErrUnauthorized).Code != "unauthorized" {
		t.Fatal("unauthorized error has wrong code")
	}
	if websocketError(os.ErrClosed).Code != "internal_error" {
		t.Fatal("internal error has wrong code")
	}

	hub := newChatHub()
	first := &chatClient{userID: 1, chats: make(map[int64]struct{})}
	second := &chatClient{userID: 1, chats: make(map[int64]struct{})}
	hub.add(first)
	hub.add(second)
	hub.mu.RLock()
	clientsCount := len(hub.users[1])
	hub.mu.RUnlock()
	if clientsCount != 2 {
		t.Fatalf("hub stored %d connections, want 2", clientsCount)
	}
	hub.remove(first)
	hub.mu.RLock()
	clientsCount = len(hub.users[1])
	hub.mu.RUnlock()
	if clientsCount != 1 {
		t.Fatalf("removing first connection left %d connections, want 1", clientsCount)
	}
	hub.remove(second)
	if _, ok := hub.users[1]; ok {
		t.Fatal("user entry was not removed after the last connection closed")
	}
}
