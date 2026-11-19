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
	uid,
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

const requireParticipantByUUIDQuery = `
	SELECT cp.profile_id
	FROM chat c
	JOIN chat_profile cp ON cp.chat_id = c.id
	WHERE c.uid = $1::uuid
		AND cp.profile_id = $2
		AND cp.deleted_at IS NULL
	FOR UPDATE OF cp`

const checkRepostAccessByUUIDQuery = `
	SELECT m.id
	FROM message m
	JOIN chat_profile cp ON cp.chat_id = m.chat_id
	WHERE m.uid = $1::uuid
		AND m.deleted_at IS NULL
		AND cp.profile_id = $2
		AND cp.deleted_at IS NULL
	FOR UPDATE OF m, cp`

const checkReplyChatByUUIDQuery = `
	SELECT m.id
	FROM message m
	WHERE m.uid = $1::uuid
		AND m.chat_id = (SELECT id FROM chat WHERE uid = $2::uuid)
		AND m.deleted_at IS NULL
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

const listMessagesByUUIDQuery = `
	SELECT ` + messageColumns + `
	FROM message
	WHERE chat_id = (SELECT id FROM chat WHERE uid = $1::uuid)
		AND deleted_at IS NULL
		AND ($2 = '' OR id < (
			SELECT id FROM message WHERE uid = NULLIF($2, '')::uuid
		))
	ORDER BY id DESC
	LIMIT $3`

const createMessageByUUIDQuery = `
	INSERT INTO message (
		chat_id,
		author_id,
		message_text,
		repost_from,
		reply_from
	)
	SELECT
		c.id,
		$2,
		$3,
		CASE WHEN $4 = '' THEN NULL ELSE (SELECT id FROM message WHERE uid = $4::uuid) END,
		CASE WHEN $5 = '' THEN NULL ELSE (SELECT id FROM message WHERE uid = $5::uuid) END
	FROM chat c
	WHERE c.uid = $1::uuid
	RETURNING ` + messageColumns

// CreateMessageParams содержит данные для создания сообщения.
type CreateMessageParams struct {
	ChatID      int64
	AuthorID    int64
	MessageText *string
	RepostFrom  *int64
	ReplyFrom   *int64
}

// CreateMessageByUUIDParams содержит внешние UUID чата и родительских сообщений.
type CreateMessageByUUIDParams struct {
	ChatUID       string
	AuthorID      int64
	MessageText   *string
	RepostFromUID string
	ReplyFromUID  string
}

// CreateByUUID создаёт сообщение, используя UUID непосредственно в SQL.
func (r *MessageRepository) CreateByUUID(ctx context.Context, params CreateMessageByUUIDParams) (domain.Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Message{}, err
	}
	defer tx.Rollback(ctx)
	var participantID int64
	if err := tx.QueryRow(ctx, requireParticipantByUUIDQuery, params.ChatUID, params.AuthorID).Scan(&participantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Message{}, ErrNotChatParticipant
		}
		return domain.Message{}, err
	}
	if params.RepostFromUID != "" {
		var sourceID int64
		if err := tx.QueryRow(ctx, checkRepostAccessByUUIDQuery, params.RepostFromUID, params.AuthorID).Scan(&sourceID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Message{}, ErrRepostForbidden
			}
			return domain.Message{}, err
		}
	}
	if params.ReplyFromUID != "" {
		var replyID int64
		if err := tx.QueryRow(ctx, checkReplyChatByUUIDQuery, params.ReplyFromUID, params.ChatUID).Scan(&replyID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.Message{}, ErrReplyForbidden
			}
			return domain.Message{}, err
		}
	}
	message, err := scanMessage(tx.QueryRow(ctx, createMessageByUUIDQuery, params.ChatUID, params.AuthorID, nullableString(params.MessageText), params.RepostFromUID, params.ReplyFromUID))
	if err != nil {
		return domain.Message{}, normalizeMessageError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Message{}, err
	}
	message.ChatUID = params.ChatUID
	if params.RepostFromUID != "" {
		value := params.RepostFromUID
		message.RepostFromUID = &value
	}
	if params.ReplyFromUID != "" {
		value := params.ReplyFromUID
		message.ReplyFromUID = &value
	}
	return message, nil
}

// ListMessagesByUUID возвращает сообщения чата без предварительного поиска его ID.
func (r *MessageRepository) ListMessagesByUUID(ctx context.Context, chatUID, cursorUID string, limit int) (domain.MessagesResponse, error) {
	limit = normalizeLimit(limit, 100)
	response := domain.MessagesResponse{Messages: make([]domain.Message, 0), Parents: make([]domain.Message, 0), ChatUID: chatUID, CursorUID: cursorUID, Limit: limit}
	rows, err := r.pool.Query(ctx, listMessagesByUUIDQuery, chatUID, cursorUID, limit+1)
	if err != nil {
		return response, err
	}
	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			rows.Close()
			return response, scanErr
		}
		message.ChatUID = chatUID
		response.Messages = append(response.Messages, message)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return response, err
	}
	rows.Close()
	response.HasMore = len(response.Messages) > limit
	if response.HasMore {
		response.Messages = response.Messages[:limit]
	}
	if len(response.Messages) > 0 {
		response.NextCursorUID = response.Messages[len(response.Messages)-1].UID
	}
	parentIDs := collectParentIDs(response.Messages)
	if len(parentIDs) > 0 {
		response.Parents, err = r.readParentMessages(ctx, parentIDs)
		if err != nil {
			return response, err
		}
		for i := range response.Parents {
			response.Parents[i].ChatUID = chatUID
		}
	}
	return response, nil
}

func scanMessage(row rowScanner) (domain.Message, error) {
	var (
		message   domain.Message
		uid       pgtype.UUID
		text      pgtype.Text
		repost    pgtype.Int8
		reply     pgtype.Int8
		createdAt pgtype.Timestamptz
		updatedAt pgtype.Timestamptz
		deletedAt pgtype.Timestamptz
	)

	err := row.Scan(
		&message.ID,
		&uid,
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
	message.UID = uuidString(uid)
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
