package domain

import "time"

// Chat описывает чат пользователя
type Chat struct {
	ID           int64     `json:"-"`
	UID          string    `json:"uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Participants []Profile `json:"participants,omitempty"`
	LastMessage  *Message  `json:"last_message,omitempty"`
	StreakDays   int       `json:"streak_days" example:"5"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// Message описывает текстовое сообщение в чате
type Message struct {
	ID            int64      `json:"-"`
	UID           string     `json:"uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID        int64      `json:"-"`
	ChatUID       string     `json:"chat_uid,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AuthorID      int64      `json:"-"`
	AuthorUID     string     `json:"author_uid,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageText   *string    `json:"message_text" example:"Привет!"`
	RepostFrom    *int64     `json:"-"`
	RepostFromUID *string    `json:"repost_from,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReplyFrom     *int64     `json:"-"`
	ReplyFromUID  *string    `json:"reply_from,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt     time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" example:"2026-01-01T12:05:00Z"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// SendMessageRequest содержит текст нового сообщения
type SendMessageRequest struct {
	ChatID        int64   `json:"-"`
	ChatUID       string  `json:"chat_uid" example:"550e8400-e29b-41d4-a716-446655440000" validate:"required"`
	MessageText   *string `json:"message_text" example:"Привет!" validate:"required,max=256"`
	RepostFrom    *int64  `json:"-"`
	RepostFromUID *string `json:"repost_from,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReplyFrom     *int64  `json:"-"`
	ReplyFromUID  *string `json:"reply_from,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ChatWebSocketAuthRequest описывает первое WebSocket-сообщение авторизации
type ChatWebSocketAuthRequest struct {
	Type      string `json:"type"`
	CSRFToken string `json:"csrf_token"`
}

// ChatWebSocketRequest описывает входящее WebSocket-событие чата
type ChatWebSocketRequest struct {
	Type    string   `json:"type"`
	ChatID  int64    `json:"-"`
	ChatUID string   `json:"chat_uid,omitempty"`
	Message *Message `json:"message,omitempty"`
}

// ChatWebSocketEvent описывает исходящее WebSocket-событие чата
type ChatWebSocketEvent struct {
	Type     string         `json:"type"`
	ChatID   int64          `json:"-"`
	ChatUID  string         `json:"chat_uid,omitempty"`
	ChatIDs  []int64        `json:"-"`
	ChatUIDs []string       `json:"chat_uids,omitempty"`
	Message  *Message       `json:"message,omitempty"`
	Error    *ErrorResponse `json:"error,omitempty"`
}

// CreateChatRequest содержит id пользователя, с которым создаётся чат
type CreateChatRequest struct {
	ProfileID  int64  `json:"-"`
	ProfileUID string `json:"profile_uid" example:"550e8400-e29b-41d4-a716-446655440000" validate:"required"`
}

// ChatsResponse содержит страницу списка чатов
type ChatsResponse struct {
	Chats         []Chat `json:"chats"`
	Cursor        int64  `json:"-"`
	CursorUID     string `json:"cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit         int    `json:"limit" example:"10"`
	HasMore       bool   `json:"has_more" example:"true"`
	NextCursor    int64  `json:"-"`
	NextCursorUID string `json:"next_cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// MessagesResponse содержит страницу сообщений чата
type MessagesResponse struct {
	Messages []Message `json:"messages"`

	// если у какого одного или нескольих сообщений есть родители
	// (RepostFrom, ReplyFrom) то они будут в этом масиве
	Parents       []Message `json:"parents"`
	ChatID        int64     `json:"-"`
	ChatUID       string    `json:"chat_uid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Cursor        int64     `json:"-"`
	CursorUID     string    `json:"cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit         int       `json:"limit" example:"10"`
	HasMore       bool      `json:"has_more" example:"true"`
	NextCursor    int64     `json:"-"`
	NextCursorUID string    `json:"next_cursor,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}
