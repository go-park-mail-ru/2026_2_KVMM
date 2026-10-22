package repository

import (
	"context"
	"errors"
	"fmt"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

var ErrChatNotFound = errors.New("chat not found")

const createChatQuery = `
	INSERT INTO chat DEFAULT VALUES
	RETURNING id`

const findPrivateChatQuery = `
	SELECT cp1.chat_id
	FROM chat_profile cp1
	JOIN chat_profile cp2 ON cp2.chat_id = cp1.chat_id
	WHERE cp1.profile_id = $1
		AND cp2.profile_id = $2
		AND (
			SELECT COUNT(*)
			FROM chat_profile cp
			WHERE cp.chat_id = cp1.chat_id
		) = CASE WHEN $1 = $2 THEN 1 ELSE 2 END
	ORDER BY cp1.chat_id
	LIMIT 1`

const upsertParticipantQuery = `
	INSERT INTO chat_profile (profile_id, chat_id, deleted_at)
	VALUES ($1, $2, NULL)
	ON CONFLICT (profile_id, chat_id)
	DO UPDATE SET deleted_at = NULL`

const acquirePrivateChatLockQuery = `
	SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`

const countActiveProfilesQuery = `
	SELECT COUNT(*)
	FROM profile
	WHERE id = ANY($1::BIGINT[])
		AND deleted_at IS NULL`

const readChatQuery = `
	SELECT id, created_at
	FROM chat
	WHERE id = $1`

const requireParticipantQuery = `
	SELECT profile_id
	FROM chat_profile
	WHERE chat_id = $1
		AND profile_id = $2
		AND deleted_at IS NULL`

const checkParticipantQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM chat_profile
		WHERE chat_id = $1
			AND profile_id = $2
			AND deleted_at IS NULL
	)`

const listChatsQuery = `
	SELECT c.id, c.created_at
	FROM chat c
	JOIN chat_profile cp ON cp.chat_id = c.id
	WHERE cp.profile_id = $1
		AND cp.deleted_at IS NULL
		AND ($2 = 0 OR c.id < $2)
	ORDER BY c.id DESC
	LIMIT $3`

const readParticipantsQuery = `
	SELECT ` + profileColumns + `
	FROM profile
	WHERE id IN (
		SELECT profile_id
		FROM chat_profile
		WHERE chat_id = $1
			AND deleted_at IS NULL
	)
	AND deleted_at IS NULL
	ORDER BY id`

const readLastMessageQuery = `
	SELECT ` + messageColumns + `
	FROM message
	WHERE chat_id = $1
		AND deleted_at IS NULL
	ORDER BY id DESC
	LIMIT 1`

// Create создаёт личный чат или возвращает существующий
// чат пользователя с самим собой тоже поддерживается
func (r *ChatRepository) Create(
	ctx context.Context,
	profileID int64,
	otherProfileID int64,
) (domain.Chat, error) {
	if profileID <= 0 || otherProfileID <= 0 {
		return domain.Chat{}, ErrProfileNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	defer tx.Rollback(ctx)

	firstID, secondID := profileID, otherProfileID
	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	lockKey := fmt.Sprintf("private-chat:%d:%d", firstID, secondID)
	if _, err = tx.Exec(ctx, acquirePrivateChatLockQuery, lockKey); err != nil {
		return domain.Chat{}, err
	}

	var profileCount int
	err = tx.QueryRow(
		ctx,
		countActiveProfilesQuery,
		[]int64{firstID, secondID},
	).Scan(&profileCount)
	if err != nil {
		return domain.Chat{}, err
	}

	expectedCount := 2
	if firstID == secondID {
		expectedCount = 1
	}
	if profileCount != expectedCount {
		return domain.Chat{}, ErrProfileNotFound
	}

	var chatID int64
	err = tx.QueryRow(
		ctx,
		findPrivateChatQuery,
		firstID,
		secondID,
	).Scan(&chatID)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Chat{}, err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, createChatQuery).Scan(&chatID)
		if err != nil {
			return domain.Chat{}, err
		}
	}

	if _, err = tx.Exec(
		ctx,
		upsertParticipantQuery,
		firstID,
		chatID,
	); err != nil {
		return domain.Chat{}, err
	}

	if firstID != secondID {
		if _, err = tx.Exec(
			ctx,
			upsertParticipantQuery,
			secondID,
			chatID,
		); err != nil {
			return domain.Chat{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return domain.Chat{}, err
	}

	var chat domain.Chat
	err = r.pool.QueryRow(ctx, readChatQuery, chatID).Scan(
		&chat.ID,
		&chat.CreatedAt,
	)
	if err != nil {
		return domain.Chat{}, err
	}

	chat.Participants, err = r.readParticipants(ctx, chatID)
	if err != nil {
		return domain.Chat{}, err
	}

	return chat, nil
}

// ListChats возвращает чаты пользователя с курсорной пагинацией по ID
func (r *ChatRepository) ListChats(
	ctx context.Context,
	profileID int64,
	cursor int64,
	limit int,
) (domain.ChatsResponse, error) {
	cursor = normalizeCursor(cursor)
	limit = normalizeLimit(limit, 100)

	if profileID <= 0 {
		return domain.ChatsResponse{}, ErrProfileNotFound
	}

	response := domain.ChatsResponse{
		Chats:  make([]domain.Chat, 0),
		Cursor: cursor,
		Limit:  limit,
	}

	rows, err := r.pool.Query(
		ctx,
		listChatsQuery,
		profileID,
		int64(cursor),
		limit+1,
	)
	if err != nil {
		return response, err
	}

	for rows.Next() {
		var chat domain.Chat
		if err = rows.Scan(&chat.ID, &chat.CreatedAt); err != nil {
			rows.Close()
			return response, err
		}
		response.Chats = append(response.Chats, chat)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return response, err
	}

	response.Chats, response.NextCursor, response.HasMore =
		paginateChats(response.Chats, limit)

	for i := range response.Chats {
		chat := &response.Chats[i]

		chat.Participants, err = r.readParticipants(ctx, chat.ID)
		if err != nil {
			return response, err
		}

		chat.LastMessage, err = r.readLastMessage(ctx, chat.ID)
		if err != nil {
			return response, err
		}
	}

	return response, nil
}

// requireParticipant проверяет, что пользователь состоит в чате
func (r *ChatRepository) IsParticipant(
	ctx context.Context,
	chatID int64,
	profileID int64,
) error {
	var id int64

	err := r.pool.QueryRow(
		ctx,
		requireParticipantQuery,
		chatID,
		profileID,
	).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotChatParticipant
	}

	return err
}

// readParticipants загружает активных участников чата
func (r *ChatRepository) readParticipants(
	ctx context.Context,
	chatID int64,
) ([]domain.Profile, error) {
	rows, err := r.pool.Query(ctx, readParticipantsQuery, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]domain.Profile, 0, 2)

	for rows.Next() {
		profile, _, scanErr := scanProfile(rows, false)
		if scanErr != nil {
			return nil, scanErr
		}
		profiles = append(profiles, profile)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

// readLastMessage возвращает последнее неудалённое сообщение чата
func (r *ChatRepository) readLastMessage(
	ctx context.Context,
	chatID int64,
) (*domain.Message, error) {
	message, err := scanMessage(
		r.pool.QueryRow(ctx, readLastMessageQuery, chatID),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &message, nil
}

// paginateChats обрабатывает результат курсорной пагинации
func paginateChats(
	chats []domain.Chat,
	limit int,
) ([]domain.Chat, int64, bool) {
	hasMore := len(chats) > limit

	if hasMore {
		chats = chats[:limit]
	}

	var nextCursor int64
	if len(chats) > 0 {
		nextCursor = chats[len(chats)-1].ID
	}

	return chats, nextCursor, hasMore
}
