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

	if err := connection.WriteJSON(domain.ChatWebSocketAuthRequest{Type: "auth", CSRFToken: "csrf", ChatIDs: []int64{1}}); err != nil {
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
	ids, err := uniquePositiveChatIDs([]int64{1, 2, 1})
	if err != nil || len(ids) != 2 || firstChatID(ids) != 1 {
		t.Fatalf("uniquePositiveChatIDs() = %v, %v", ids, err)
	}
	if _, err := uniquePositiveChatIDs([]int64{0}); err == nil {
		t.Fatal("uniquePositiveChatIDs() expected invalid id error")
	}
	if _, err := uniquePositiveChatIDs(nil); err == nil {
		t.Fatal("uniquePositiveChatIDs(empty) expected error")
	}

	if websocketError(errInvalidChatID).Code != "invalid_request" {
		t.Fatal("invalid chat error has wrong code")
	}
	if websocketError(chusecase.ErrUnauthorized).Code != "unauthorized" {
		t.Fatal("unauthorized error has wrong code")
	}
	if websocketError(os.ErrClosed).Code != "internal_error" {
		t.Fatal("internal error has wrong code")
	}

	hub := newChatHub()
	first := &chatClient{userID: 1, chats: make(map[int64]struct{})}
	second := &chatClient{userID: 1, chats: make(map[int64]struct{})}
	previous := hub.add(first)
	if previous != nil {
		t.Fatal("first hub add returned previous client")
	}
	if hub.add(second) != first {
		t.Fatal("second hub add did not return previous client")
	}
	hub.remove(first)
	hub.remove(second)
}
