package repository

import (
	"context"
	"errors"
	"fmt"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
)

type ChatRepository struct {
	pool dbPool
}

func NewChatRepository(pool dbPool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

var (
	ErrChatNotFound    = errors.New("chat not found")
	ErrRepostForbidden = errors.New("repost source message is not accessible")
	ErrReplyForbidden  = errors.New("reply message does not belong to target chat")
)

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

const upsertParticipantsQuery = `
	INSERT INTO chat_profile (profile_id, chat_id, deleted_at)
	SELECT participant.profile_id, $2, NULL
	FROM unnest($1::BIGINT[]) AS participant(profile_id)
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

const checkRepostAccessQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM message m
		JOIN chat_profile cp ON cp.chat_id = m.chat_id
		WHERE m.id = $1
			AND m.deleted_at IS NULL
			AND cp.profile_id = $2
			AND cp.deleted_at IS NULL
	)`

const checkReplyChatQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM message
		WHERE id = $1
			AND chat_id = $2
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

const readParticipantsBatchQuery = `
	SELECT
		cp.chat_id,
		p.id,
		p.nickname,
		p.email,
		p.phone_number,
		p.profile_name,
		p.surname,
		p.patronymic,
		p.gender,
		p.birthday,
		p.bio,
		p.created_at,
		p.updated_at,
		p.deleted_at
	FROM chat_profile cp
	JOIN profile p ON p.id = cp.profile_id
	WHERE cp.chat_id = ANY($1::BIGINT[])
		AND cp.deleted_at IS NULL
		AND p.deleted_at IS NULL
	ORDER BY cp.chat_id, p.id`

const readLastMessagesBatchQuery = `
	SELECT DISTINCT ON (m.chat_id) ` + messageColumns + `
	FROM message m
	WHERE m.chat_id = ANY($1::BIGINT[])
		AND m.deleted_at IS NULL
	ORDER BY m.chat_id, m.id DESC`

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

	participantIDs := []int64{firstID}
	if firstID != secondID {
		participantIDs = append(participantIDs, secondID)
	}
	if _, err = tx.Exec(ctx, upsertParticipantsQuery, participantIDs, chatID); err != nil {
		return domain.Chat{}, err
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

	chatIDs := make([]int64, 0, len(response.Chats))
	for _, chat := range response.Chats {
		chatIDs = append(chatIDs, chat.ID)
	}

	participants, err := r.readParticipantsBatch(ctx, chatIDs)
	if err != nil {
		return response, err
	}
	lastMessages, err := r.readLastMessagesBatch(ctx, chatIDs)
	if err != nil {
		return response, err
	}

	for i := range response.Chats {
		chat := &response.Chats[i]
		chat.Participants = participants[chat.ID]
		chat.LastMessage = lastMessages[chat.ID]
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

// ValidateMessageReferences проверяет доступ пользователя к чату,
// исходному чату пересылаемого сообщения и соответствие сообщения для ответа
func (r *ChatRepository) ValidateMessageReferences(
	ctx context.Context,
	userID int64,
	targetChatID int64,
	repostFrom *int64,
	replyFrom *int64,
) error {
	if err := r.IsParticipant(ctx, targetChatID, userID); err != nil {
		return err
	}

	if repostFrom != nil {
		var allowed bool
		if err := r.pool.QueryRow(
			ctx,
			checkRepostAccessQuery,
			*repostFrom,
			userID,
		).Scan(&allowed); err != nil {
			return err
		}
		if !allowed {
			return ErrRepostForbidden
		}
	}

	if replyFrom != nil {
		var belongsToChat bool
		if err := r.pool.QueryRow(
			ctx,
			checkReplyChatQuery,
			*replyFrom,
			targetChatID,
		).Scan(&belongsToChat); err != nil {
			return err
		}
		if !belongsToChat {
			return ErrReplyForbidden
		}
	}

	return nil
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

// readParticipantsBatch загружает участников всех чатов одним запросом
func (r *ChatRepository) readParticipantsBatch(
	ctx context.Context,
	chatIDs []int64,
) (map[int64][]domain.Profile, error) {
	participants := make(map[int64][]domain.Profile, len(chatIDs))
	if len(chatIDs) == 0 {
		return participants, nil
	}

	rows, err := r.pool.Query(ctx, readParticipantsBatchQuery, chatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chatID int64
		profile, _, scanErr := scanProfileWithLeading(rows, &chatID, false)
		if scanErr != nil {
			return nil, scanErr
		}
		participants[chatID] = append(participants[chatID], profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return participants, nil
}

// readLastMessagesBatch загружает последние неудалённые сообщения всех чатов одним запросом
func (r *ChatRepository) readLastMessagesBatch(
	ctx context.Context,
	chatIDs []int64,
) (map[int64]*domain.Message, error) {
	lastMessages := make(map[int64]*domain.Message, len(chatIDs))
	if len(chatIDs) == 0 {
		return lastMessages, nil
	}

	rows, err := r.pool.Query(ctx, readLastMessagesBatchQuery, chatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		message, scanErr := scanMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		lastMessages[message.ChatID] = &message
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lastMessages, nil
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
