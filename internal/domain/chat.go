package domain

import "time"

// Chat описывает чат пользователя
type Chat struct {
	ID           int64     `json:"id" example:"1"`
	Participants []Profile `json:"participants,omitempty"`
	LastMessage  *Message  `json:"last_message,omitempty"`
	CreatedAt    time.Time `json:"created_at" example:"2026-01-01T12:00:00Z"`
}

// Message описывает текстовое сообщение в чате
type Message struct {
	ID          int64      `json:"id" example:"1"`
	ChatID      int64      `json:"chat_id" example:"1"`
	AuthorID    int64      `json:"author_id" example:"1"`
	MessageText string     `json:"message_text" example:"Привет!"`
	RepostFrom  *int64     `json:"repost_from,omitempty" example:"5"`
	ReplyFrom   *int64     `json:"reply_from,omitempty" example:"7"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-01-01T12:00:00Z"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" example:"2026-01-01T12:05:00Z"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// SendMessageRequest содержит текст нового сообщения
type SendMessageRequest struct {
	ChatID      int64  `json:"chat_id" example:"1" validate:"required,gt=0"`
	MessageText string `json:"message_text" example:"Привет!" validate:"required,max=256"`
	RepostFrom  *int64 `json:"repost_from,omitempty" example:"5" validate:"omitempty,gt=0"`
	ReplyFrom   *int64 `json:"reply_from,omitempty" example:"7" validate:"omitempty,gt=0"`
}

// CreateChatRequest содержит id пользователя, с которым создаётся чат
type CreateChatRequest struct {
	ProfileID int64 `json:"profile_id" example:"2" validate:"required,gt=0"`
}

// ChatsResponse содержит страницу списка чатов
type ChatsResponse struct {
	Chats      []Chat `json:"chats"`
	Offset     int    `json:"offset" example:"0"`
	Limit      int    `json:"limit" example:"10"`
	HasMore    bool   `json:"has_more" example:"true"`
	NextOffset int    `json:"next_offset,omitempty" example:"10"`
}

// MessagesResponse содержит страницу сообщений чата
type MessagesResponse struct {
	Messages []Message `json:"messages"`

	// если у какого одного или нескольих сообщений есть родители
	// (RepostFrom, ReplyFrom) то они будут в этом масиве
	Parents    []Message `json:"parents"`
	ChatID     int64     `json:"chat_id" example:"1"`
	Offset     int       `json:"offset" example:"0"`
	Limit      int       `json:"limit" example:"10"`
	HasMore    bool      `json:"has_more" example:"true"`
	NextOffset int       `json:"next_offset,omitempty" example:"10"`
}
