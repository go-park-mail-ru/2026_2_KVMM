package chat

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/handler/helpers"
	"kvmm/internal/logging"
	chusecase "kvmm/internal/usecase/chat"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	websocketMessageLimit  = 64 * 1024
	websocketChatPageLimit = 100
)

var chatWebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     allowedWebSocketOrigin,
}

var (
	errChatNotSubscribed = errors.New("chat is not subscribed")
	errUnknownEvent      = errors.New("unknown WebSocket event type")
)

type chatClient struct {
	userID int64
	conn   *websocket.Conn

	writeMu  sync.Mutex
	chatsMu  sync.RWMutex
	chats    map[int64]struct{}
	chatUIDs map[string]int64
}

type chatHub struct {
	mu    sync.RWMutex
	users map[int64]map[*chatClient]struct{}
}

// newChatHub создаёт хранилище WebSocket-подключений пользователей
func newChatHub() *chatHub {
	return &chatHub{users: make(map[int64]map[*chatClient]struct{})}
}

// add добавляет новое WebSocket-подключение пользователя
func (h *chatHub) add(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.users[client.userID]
	if clients == nil {
		clients = make(map[*chatClient]struct{})
		h.users[client.userID] = clients
	}
	clients[client] = struct{}{}
}

// remove удаляет конкретное WebSocket-подключение пользователя.
func (h *chatHub) remove(client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.users[client.userID]
	if clients == nil {
		return
	}
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.users, client.userID)
	}
}

// broadcast отправляет событие подписанным на чат пользователям
func (h *chatHub) broadcast(chatID int64, event domain.ChatWebSocketEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	h.mu.RLock()
	clients := make([]*chatClient, 0)
	for _, userClients := range h.users {
		for client := range userClients {
			if client.subscribed(chatID) {
				clients = append(clients, client)
			}
		}
	}
	h.mu.RUnlock()

	var broadcastErr error
	for _, client := range clients {
		if err := client.write(payload); err != nil {
			h.remove(client)
			if closeErr := client.conn.Close(); closeErr != nil {
				broadcastErr = errors.Join(broadcastErr, err, closeErr)
				continue
			}
			broadcastErr = errors.Join(broadcastErr, err)
		}
	}
	return broadcastErr
}

// subscribe добавляет чат в список подписок пользователя
func (c *chatClient) subscribe(chatID int64) {
	c.chatsMu.Lock()
	defer c.chatsMu.Unlock()
	c.chats[chatID] = struct{}{}
}

// subscribeUID добавляет чат во внутреннюю карту подписок по UUID
func (c *chatClient) subscribeUID(uid string, chatID int64) {
	c.chatsMu.Lock()
	defer c.chatsMu.Unlock()
	c.chats[chatID] = struct{}{}
	c.chatUIDs[uid] = chatID
}

// chatIDByUID возвращает внутренний ID подписанного чата.
func (c *chatClient) chatIDByUID(uid string) (int64, bool) {
	c.chatsMu.RLock()
	defer c.chatsMu.RUnlock()
	chatID, ok := c.chatUIDs[uid]
	return chatID, ok
}

// subscribed проверяет подписку пользователя на чат
func (c *chatClient) subscribed(chatID int64) bool {
	c.chatsMu.RLock()
	defer c.chatsMu.RUnlock()
	_, ok := c.chats[chatID]
	return ok
}

// write отправляет одно сообщение через WebSocket без параллельных записей
func (c *chatClient) write(payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

// WebSocket открывает WebSocket-соединение пользователя для всех его чатов
// Один пользователь может одновременно подключиться с нескольких устройств
// WebSocket godoc
// @Summary Подключение к чатам через WebSocket
// @Description Выполняет WebSocket upgrade. Session cookie передаётся автоматически. Первым сообщением необходимо отправить `{"type":"auth","csrf_token":"TOKEN"}`. После авторизации сервер загружает все чаты пользователя через пагинацию и подписывает соединение на них. Один аккаунт может иметь несколько одновременных подключений с разных устройств. Сообщение отправляется как `{"type":"message","chat_uid":"UUID","message":{"message_text":"Привет!"}}`. Ошибки авторизации, CSRF и проверки доступа после Upgrade отправляются как WebSocket-события с `type: "error"`.
// @Tags chats
// @Security SessionCookie
// @Produce json
// @Success 101 {string} string "WebSocket connection established"
// @Failure 400 {string} string "Invalid WebSocket handshake"
// @Failure 403 {string} string "WebSocket origin is not allowed"
// @Router /api/chats/ws [get]
func (h *ChatHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	sessionToken := helpers.SessionCookie(r)
	conn, err := chatWebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Logger().Error("WebSocket upgrade failed",
			zap.Error(err),
			zap.String("origin", r.Header.Get("Origin")),
			zap.String("path", r.URL.Path),
		)
		return
	}
	defer conn.Close()
	conn.SetReadLimit(websocketMessageLimit)
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return
	}

	var authRequest domain.ChatWebSocketAuthRequest
	if err := readWebSocketJSON(conn, &authRequest); err != nil || authRequest.Type != "auth" {
		if err := writeWebSocketEvent(conn, domain.ChatWebSocketEvent{
			Type: "error",
			Error: &domain.ErrorResponse{
				Code:    "authentication_required",
				Message: "the first WebSocket message must authenticate the connection",
			},
		}); err != nil {
			return
		}
		return
	}

	userID, err := h.sessions.Authorize(sessionToken, authRequest.CSRFToken)
	if err != nil {
		if err := writeWebSocketEvent(conn, domain.ChatWebSocketEvent{
			Type:  "error",
			Error: websocketError(err),
		}); err != nil {
			return
		}
		return
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		return
	}
	csrfToken := authRequest.CSRFToken

	client := &chatClient{
		userID:   userID,
		conn:     conn,
		chats:    make(map[int64]struct{}),
		chatUIDs: make(map[string]int64),
	}
	h.hub.add(client)
	defer h.hub.remove(client)

	if err := h.writeClientEvent(client, domain.ChatWebSocketEvent{Type: "connected"}); err != nil {
		return
	}
	if err := h.subscribeAllChats(r, client, sessionToken, csrfToken); err != nil {
		return
	}

	for {
		var request domain.ChatWebSocketRequest
		if err := readWebSocketJSON(conn, &request); err != nil {
			logging.Logger().Debug("WebSocket request read failed", zap.Error(err))
			_ = h.writeClientEvent(client, domain.ChatWebSocketEvent{
				Type:  "error",
				Error: &domain.ErrorResponse{Code: "bad_request", Message: "invalid WebSocket request"},
			})
			return
		}

		switch request.Type {
		case "message", "":
			if err := h.sendMessage(r, client, sessionToken, csrfToken, request); err != nil {
				return
			}
		default:
			if err := h.writeWebSocketError(client, request.ChatUID, errUnknownEvent); err != nil {
				return
			}
		}
	}
}

// subscribeAllChats загружает все чаты пользователя и добавляет их в подписки соединения
func (h *ChatHandler) subscribeAllChats(
	r *http.Request,
	client *chatClient,
	sessionToken string,
	csrfToken string,
) error {
	cursor := ""
	chatUIDs := make([]string, 0)
	for {
		response, err := h.useCase.ListByUID(
			r.Context(),
			sessionToken,
			csrfToken,
			cursor,
			websocketChatPageLimit,
		)
		if err != nil {
			return h.writeWebSocketError(client, "", err)
		}

		for _, chat := range response.Chats {
			if chat.ID > 0 {
				chatUIDs = append(chatUIDs, chat.UID)
				client.subscribeUID(chat.UID, chat.ID)
			}
		}

		if !response.HasMore {
			break
		}
		if response.NextCursorUID == "" || response.NextCursorUID == cursor {
			return h.writeWebSocketError(client, "", errors.New("invalid chats pagination"))
		}
		cursor = response.NextCursorUID
	}

	event := domain.ChatWebSocketEvent{Type: "subscribed"}
	if len(chatUIDs) == 1 {
		event.ChatUID = chatUIDs[0]
	} else {
		event.ChatUIDs = chatUIDs
	}
	return h.writeClientEvent(client, event)
}

// sendMessage отправляет сообщение и публикует его подписанным пользователям
func (h *ChatHandler) sendMessage(
	r *http.Request,
	client *chatClient,
	sessionToken string,
	csrfToken string,
	request domain.ChatWebSocketRequest,
) error {
	if !helpers.IsUUID(request.ChatUID) {
		return h.writeWebSocketError(client, request.ChatUID, errors.New("chat_uid must be a valid UUID"))
	}
	chatID, ok := client.chatIDByUID(request.ChatUID)
	if !ok {
		return h.writeWebSocketError(client, request.ChatUID, errChatNotSubscribed)
	}

	sendRequest := domain.SendMessageRequest{ChatUID: request.ChatUID}
	if request.Message != nil {
		sendRequest.MessageText = request.Message.MessageText
		sendRequest.RepostFrom = request.Message.RepostFrom
		sendRequest.ReplyFrom = request.Message.ReplyFrom
		sendRequest.RepostFromUID = request.Message.RepostFromUID
		sendRequest.ReplyFromUID = request.Message.ReplyFromUID
	}
	message, err := h.useCase.SendByUID(r.Context(), sessionToken, csrfToken, sendRequest)
	if err != nil {
		return h.writeWebSocketError(client, request.ChatUID, err)
	}

	if err := h.hub.broadcast(chatID, domain.ChatWebSocketEvent{
		Type:    "message",
		ChatUID: request.ChatUID,
		Message: &message,
	}); err != nil {
		return h.writeWebSocketError(client, request.ChatUID, err)
	}
	return nil
}

// readWebSocketJSON декодирует событие и отклоняет неизвестные поля
func readWebSocketJSON(conn *websocket.Conn, value any) error {
	_, reader, err := conn.NextReader()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

// writeWebSocketError отправляет ошибку в формате WebSocket-события
func (h *ChatHandler) writeWebSocketError(client *chatClient, chatUID string, err error) error {
	return h.writeClientEvent(client, domain.ChatWebSocketEvent{
		Type:    "error",
		ChatUID: chatUID,
		Error:   websocketError(err),
	})
}

// writeClientEvent отправляет событие и отключает клиента при ошибке записи
func (h *ChatHandler) writeClientEvent(client *chatClient, event domain.ChatWebSocketEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err = client.write(payload); err == nil {
		return nil
	}
	logging.Logger().Error("WebSocket event write failed", zap.Error(err))
	h.hub.remove(client)
	if closeErr := client.conn.Close(); closeErr != nil {
		return errors.Join(err, closeErr)
	}
	return err
}

func allowedWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = os.Getenv("FRONTEND_URL")
	}
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8001"
	}
	return allowedOrigin == "*" || origin == allowedOrigin
}

// websocketError преобразует ошибку usecase в формат WebSocket-события
func websocketError(err error) *domain.ErrorResponse {
	response := &domain.ErrorResponse{
		Code:    "invalid_request",
		Message: err.Error(),
	}
	switch {
	case errors.Is(err, chusecase.ErrUnauthorized):
		response.Code = "unauthorized"
	case errors.Is(err, chusecase.ErrInvalidCSRF):
		response.Code = "csrf_failed"
	case errors.Is(err, chusecase.ErrChatNotFound):
		response.Code = "chat_not_found"
		response.Message = "chat not found"
	case errors.Is(err, chusecase.ErrNotParticipant):
		response.Code = "not_chat_participant"
		response.Message = "user is not a chat participant"
	case errors.Is(err, chusecase.ErrEmptyMessage), errors.Is(err, chusecase.ErrMessageTooLong),
		errors.Is(err, errChatNotSubscribed), errors.Is(err, errUnknownEvent):
		return response
	default:
		logging.Logger().Error("internal WebSocket error", zap.Error(err))
		response.Code = "internal_error"
		response.Message = "internal server error"
	}
	return response
}

// writeWebSocketEvent отправляет событие до добавления соединения в hub
func writeWebSocketEvent(conn *websocket.Conn, event domain.ChatWebSocketEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}
