package repository

import (
	"context"
	"errors"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrNotChatParticipant = errors.New("user is not a chat participant")
	ErrRepostForbidden    = errors.New("repost source message is not accessible")
	ErrReplyForbidden     = errors.New("reply message does not belong to target chat")
	ErrEmptyMessage       = errors.New("message text is empty")
	ErrMessageTooLong     = errors.New("message text exceeds 256 characters")
)

type MessageRepository struct {
	pool dbPool
}

func NewMessageRepository(pool dbPool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

const messageColumns = `
	id,
	chat_id,
	author_id,
	message_text,
	repost_from,
	reply_from,
	created_at,
	updated_at,
	deleted_at`

const createMessageQuery = `
	INSERT INTO message (
		chat_id,
		author_id,
		message_text,
		repost_from,
		reply_from
	)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING ` + messageColumns

const requireParticipantQuery = `
	SELECT profile_id
	FROM chat_profile
	WHERE chat_id = $1
		AND profile_id = $2
		AND deleted_at IS NULL
	FOR UPDATE`

const checkRepostAccessQuery = `
	SELECT m.id
	FROM message m
	JOIN chat_profile cp ON cp.chat_id = m.chat_id
	WHERE m.id = $1
		AND m.deleted_at IS NULL
		AND cp.profile_id = $2
		AND cp.deleted_at IS NULL
	FOR UPDATE OF m, cp`

const checkReplyChatQuery = `
	SELECT id
	FROM message
	WHERE id = $1
		AND chat_id = $2
		AND deleted_at IS NULL
	FOR UPDATE`

const listMessagesQuery = `
	SELECT ` + messageColumns + `
	FROM message
	WHERE chat_id = $1
		AND deleted_at IS NULL
		AND ($2 = 0 OR id < $2)
	ORDER BY id DESC
	LIMIT $3`

const parentMessagesQuery = `
	SELECT ` + messageColumns + `
	FROM message
	WHERE id = ANY($1::BIGINT[])
		AND deleted_at IS NULL
	ORDER BY id`

// CreateMessageParams содержит данные для создания сообщения.
type CreateMessageParams struct {
	ChatID      int64
	AuthorID    int64
	MessageText *string
	RepostFrom  *int64
	ReplyFrom   *int64
}

func scanMessage(row rowScanner) (domain.Message, error) {
	var (
		message   domain.Message
		text      pgtype.Text
		repost    pgtype.Int8
		reply     pgtype.Int8
		createdAt pgtype.Timestamptz
		updatedAt pgtype.Timestamptz
		deletedAt pgtype.Timestamptz
	)

	err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.AuthorID,
		&text,
		&repost,
		&reply,
		&createdAt,
		&updatedAt,
		&deletedAt,
	)
	if err != nil {
		return domain.Message{}, err
	}

	message.MessageText = textPointer(text)
	message.RepostFrom = int64Pointer(repost)
	message.ReplyFrom = int64Pointer(reply)
	message.CreatedAt = createdAt.Time
	message.UpdatedAt = timestampPointer(updatedAt)
	message.DeletedAt = timestampPointer(deletedAt)

	return message, nil
}

// Create проверяет родителей сообщения и создаёт его в одной транзакции
func (r *MessageRepository) Create(
	ctx context.Context,
	createParam CreateMessageParams,
) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, err
	}
	defer tx.Rollback(ctx)

	var participantID int64
	if err := tx.QueryRow(ctx, requireParticipantQuery, createParam.ChatID, createParam.AuthorID).Scan(&participantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, ErrNotChatParticipant
		}
		return domain.Message{}, err
	}

	if createParam.RepostFrom != nil {
		var sourceMessageID int64
		if err := tx.QueryRow(ctx, checkRepostAccessQuery, *createParam.RepostFrom, createParam.AuthorID).Scan(&sourceMessageID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Message{}, ErrRepostForbidden
			}
			return domain.Message{}, err
		}
	}

	if createParam.ReplyFrom != nil {
		var replyMessageID int64
		if err := tx.QueryRow(ctx, checkReplyChatQuery, *createParam.ReplyFrom, createParam.ChatID).Scan(&replyMessageID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Message{}, ErrReplyForbidden
			}
			return domain.Message{}, err
		}
	}

	row := tx.QueryRow(
		ctx,
		createMessageQuery,

		createParam.ChatID,
		createParam.AuthorID,
		nullableString(createParam.MessageText),
		nullableInt64(createParam.RepostFrom),
		nullableInt64(createParam.ReplyFrom),
	)

	message, err := scanMessage(row)
	if err != nil {
		return domain.Message{}, normalizeMessageError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, err
	}
	return message, nil
}

// ListMessages возвращает страницу сообщений и их родительские сообщения
func (r *MessageRepository) ListMessages(
	ctx context.Context,
	chatID int64,
	cursor int64,
	limit int,
) (domain.MessagesResponse, error) {
	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 100)

	if chatID <= 0 {
		return domain.MessagesResponse{}, ErrChatNotFound
	}

	response := domain.MessagesResponse{
		ChatID:   chatID,
		Cursor:   cursor,
		Limit:    limit,
		Messages: make([]domain.Message, 0),
		Parents:  make([]domain.Message, 0),
	}

	rows, err := r.pool.Query(
		ctx,
		listMessagesQuery,
		chatID,
		int64(cursor),
		limit+1,
	)
	if err != nil {
		return response, err
	}

	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return response, scanErr
		}
		response.Messages = append(response.Messages, message)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return response, err
	}

	response.Messages, response.NextCursor, response.HasMore =
		paginateMessages(response.Messages, limit)

	parentIDs := collectParentIDs(response.Messages)
	if len(parentIDs) > 0 {
		response.Parents, err = r.readParentMessages(ctx, parentIDs)
		if err != nil {
			return response, err
		}
	}

	return response, nil
}

// paginateMessages обрабатывает результат курсорной пагинации
func paginateMessages(
	messages []domain.Message,
	limit int,
) ([]domain.Message, int64, bool) {
	hasMore := len(messages) > limit

	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor int64
	if len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	return messages, nextCursor, hasMore
}

// readParentMessages загружает родительские сообщения
func (r *MessageRepository) readParentMessages(
	ctx context.Context,
	parentIDs []int64,
) ([]domain.Message, error) {
	rows, err := r.pool.Query(
		ctx,
		parentMessagesQuery,
		parentIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]domain.Message, 0, len(parentIDs))

	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// collectParentIDs собирает уникальные ID родительских сообщений
func collectParentIDs(messages []domain.Message) []int64 {
	seen := make(map[int64]bool)
	ids := make([]int64, 0)

	for _, message := range messages {
		for _, id := range []*int64{
			message.RepostFrom,
			message.ReplyFrom,
		} {
			if id == nil || seen[*id] {
				continue
			}

			seen[*id] = true
			ids = append(ids, *id)
		}
	}

	return ids
}

func normalizeMessageError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMessageNotFound
	}
	return err
}
