package chat

import (
	"context"
	"strings"

	"kvmm/internal/domain"
	"kvmm/internal/logging"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

var (
	ErrUnauthorized   = sessionusecase.ErrUnauthorized
	ErrInvalidCSRF    = sessionusecase.ErrInvalidCSRF
	ErrChatNotFound   = repository.ErrChatNotFound
	ErrNotParticipant = repository.ErrNotChatParticipant
	ErrEmptyMessage   = repository.ErrEmptyMessage
	ErrMessageTooLong = repository.ErrMessageTooLong
)

type chatRepository interface {
	Create(context.Context, int64, int64) (domain.Chat, error)
	CreateByUUID(context.Context, int64, string) (domain.Chat, error)
	ListChatsByUUID(context.Context, int64, string, int) (domain.ChatsResponse, error)
	IsParticipantByUUID(context.Context, string, int64) error
	ListChats(context.Context, int64, int64, int) (domain.ChatsResponse, error)
	IsParticipant(context.Context, int64, int64) error
}

type messageRepository interface {
	Create(context.Context, repository.CreateMessageParams) (domain.Message, error)
	ListMessages(context.Context, int64, int64, int) (domain.MessagesResponse, error)
	ListMessagesByUUID(context.Context, string, string, int) (domain.MessagesResponse, error)
	CreateByUUID(context.Context, repository.CreateMessageByUUIDParams) (domain.Message, error)
}

// UseCase содержит бизнес-логику личных чатов и сообщений
type UseCase struct {
	chats    chatRepository
	messages messageRepository
	sessions *sessionusecase.UseCase
}

// New создаёт usecase чатов
func New(chats chatRepository, messages messageRepository, sessions *sessionusecase.UseCase) *UseCase {
	return &UseCase{chats: chats, messages: messages, sessions: sessions}
}

// Create создаёт личный чат с указанным пользователем
func (u *UseCase) Create(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.CreateChatRequest,
) (domain.Chat, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Chat{}, err
	}
	chat, err := u.chats.Create(ctx, userID, req.ProfileID)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "create chat", err)
	}
	return chat, err
}

// CreateByUID создаёт личный чат с пользователем, указанным внешним UUID
func (u *UseCase) CreateByUID(ctx context.Context, sessionToken, csrfToken string, req domain.CreateChatRequest) (domain.Chat, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Chat{}, err
	}
	chat, err := u.chats.CreateByUUID(ctx, userID, req.ProfileUID)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "create chat", err)
	}
	return chat, err
}

// List возвращает чаты текущего пользователя
func (u *UseCase) List(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	cursor int64,
	limit int,
) (domain.ChatsResponse, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.ChatsResponse{}, err
	}
	response, err := u.chats.ListChats(ctx, userID, cursor, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "list chats", err)
	}
	return response, err
}

// ListByUID возвращает чаты с UUID/курсором
func (u *UseCase) ListByUID(ctx context.Context, sessionToken, csrfToken, cursorUID string, limit int) (domain.ChatsResponse, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.ChatsResponse{}, err
	}
	response, err := u.chats.ListChatsByUUID(ctx, userID, cursorUID, limit)
	if err != nil {
		return domain.ChatsResponse{}, err
	}
	response.CursorUID = cursorUID
	if len(response.Chats) > 0 {
		response.NextCursorUID = response.Chats[len(response.Chats)-1].UID
	}
	return response, nil
}

// Messages возвращает сообщения чата и связанные родительские сообщения
func (u *UseCase) Messages(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	chatID int64,
	cursor int64,
	limit int,
) (domain.MessagesResponse, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.MessagesResponse{}, err
	}
	if err := u.chats.IsParticipant(ctx, chatID, userID); err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "check chat participant", err)
		return domain.MessagesResponse{}, err
	}
	response, err := u.messages.ListMessages(ctx, chatID, cursor, limit)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "list messages", err)
	}
	return response, err
}

// MessagesByUID возвращает сообщения чата с UUID/курсором
func (u *UseCase) MessagesByUID(ctx context.Context, sessionToken, csrfToken, chatUID, cursorUID string, limit int) (domain.MessagesResponse, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.MessagesResponse{}, err
	}
	if err := u.chats.IsParticipantByUUID(ctx, chatUID, userID); err != nil {
		return domain.MessagesResponse{}, err
	}
	response, err := u.messages.ListMessagesByUUID(ctx, chatUID, cursorUID, limit)
	if err != nil {
		return domain.MessagesResponse{}, err
	}
	return response, nil
}

// Send отправляет сообщение в чат
func (u *UseCase) Send(
	ctx context.Context,
	sessionToken string,
	csrfToken string,
	req domain.SendMessageRequest,
) (domain.Message, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Message{}, err
	}
	if req.MessageText == nil || strings.TrimSpace(*req.MessageText) == "" {
		return domain.Message{}, repository.ErrEmptyMessage
	}
	if len([]rune(*req.MessageText)) > 256 {
		return domain.Message{}, repository.ErrMessageTooLong
	}
	message, err := u.messages.Create(ctx, repository.CreateMessageParams{
		ChatID:      req.ChatID,
		AuthorID:    userID,
		MessageText: req.MessageText,
		RepostFrom:  req.RepostFrom,
		ReplyFrom:   req.ReplyFrom,
	})
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "send message", err)
	}
	return message, err
}

// SendByUID отправляет сообщение, используя внешние UUID чата и сообщений.
func (u *UseCase) SendByUID(ctx context.Context, sessionToken, csrfToken string, req domain.SendMessageRequest) (domain.Message, error) {
	userID, err := u.sessions.Authorize(sessionToken, csrfToken)
	if err != nil {
		return domain.Message{}, err
	}
	if req.MessageText == nil || strings.TrimSpace(*req.MessageText) == "" {
		return domain.Message{}, repository.ErrEmptyMessage
	}
	if len([]rune(*req.MessageText)) > 256 {
		return domain.Message{}, repository.ErrMessageTooLong
	}
	params := repository.CreateMessageByUUIDParams{ChatUID: req.ChatUID, AuthorID: userID, MessageText: req.MessageText}
	if req.RepostFromUID != nil {
		params.RepostFromUID = *req.RepostFromUID
	}
	if req.ReplyFromUID != nil {
		params.ReplyFromUID = *req.ReplyFromUID
	}
	message, err := u.messages.CreateByUUID(ctx, params)
	if err != nil {
		logging.LogUseCaseErrorContext(ctx, "chat", "send message", err)
	}
	return message, err
}
