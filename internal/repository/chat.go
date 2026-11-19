package repository

import (
	"context"
	"errors"
	"fmt"

	"kvmm/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChatRepository struct {
	pool dbPool
}

func NewChatRepository(pool dbPool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

var (
	ErrChatNotFound = errors.New("chat not found")
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

const countActiveProfilesByUUIDForChatQuery = `
	SELECT COUNT(*)
	FROM profile
	WHERE deleted_at IS NULL
		AND (id = $1 OR uid = $2::uuid)`

const findPrivateChatByUUIDQuery = `
	SELECT cp1.chat_id
	FROM chat_profile cp1
	JOIN chat_profile cp2 ON cp2.chat_id = cp1.chat_id
	WHERE cp1.profile_id = $1
		AND cp2.profile_id = (
			SELECT id FROM profile WHERE uid = $2::uuid AND deleted_at IS NULL
		)
		AND (
			SELECT COUNT(*)
			FROM chat_profile cp
			WHERE cp.chat_id = cp1.chat_id
		) = 2
	ORDER BY cp1.chat_id
	LIMIT 1`

const upsertParticipantsByUUIDQuery = `
	INSERT INTO chat_profile (profile_id, chat_id, deleted_at)
	SELECT p.id, $2, NULL
	FROM profile p
	WHERE p.deleted_at IS NULL
		AND (p.id = $1 OR p.uid = $3::uuid)
	ON CONFLICT (profile_id, chat_id)
	DO UPDATE SET deleted_at = NULL`

const readChatQuery = `
	SELECT id, uid, created_at
	FROM chat
	WHERE id = $1`

const checkParticipantQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM chat_profile
		WHERE chat_id = $1
			AND profile_id = $2
			AND deleted_at IS NULL
	)`

const listChatsQuery = `
	SELECT c.id, c.uid, c.created_at
	FROM chat c
	JOIN chat_profile cp ON cp.chat_id = c.id
	WHERE cp.profile_id = $1
		AND cp.deleted_at IS NULL
		AND ($2 = 0 OR c.id < $2)
	ORDER BY c.id DESC
	LIMIT $3`

const listChatsByUUIDQuery = `
	SELECT c.id, c.uid, c.created_at
	FROM chat c
	JOIN chat_profile cp ON cp.chat_id = c.id
	WHERE cp.profile_id = $1
		AND cp.deleted_at IS NULL
		AND ($2 = '' OR c.id < (
			SELECT id FROM chat WHERE uid = NULLIF($2, '')::uuid
		))
	ORDER BY c.id DESC
	LIMIT $3`

const checkParticipantByUUIDQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM chat c
		JOIN chat_profile cp ON cp.chat_id = c.id
		WHERE c.uid = $1::uuid
			AND cp.profile_id = $2
			AND cp.deleted_at IS NULL
	)`

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
		p.deleted_at,
		p.uid,
		avatar.media_path AS avatar_url
	FROM chat_profile cp
	JOIN profile p ON p.id = cp.profile_id
	LEFT JOIN LATERAL (
		SELECT m.media_path
		FROM profile_avatar pa
		JOIN media m ON m.id = pa.media_id
		WHERE pa.profile_id = p.id
			AND pa.deleted_at IS NULL
			AND m.deleted_at IS NULL
		ORDER BY pa.created_at DESC, pa.media_id DESC
		LIMIT 1
	) avatar ON TRUE
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
	var uid pgtype.UUID
	err = r.pool.QueryRow(ctx, readChatQuery, chatID).Scan(
		&chat.ID,
		&uid,
		&chat.CreatedAt,
	)
	if err != nil {
		return domain.Chat{}, err
	}
	chat.UID = uuidString(uid)

	chat.Participants, err = r.readParticipants(ctx, chatID)
	if err != nil {
		return domain.Chat{}, err
	}

	return chat, nil
}

// CreateByUUID создаёт личный чат с пользователем, указанным внешним UUID
func (r *ChatRepository) CreateByUUID(ctx context.Context, profileID int64, profileUID string) (domain.Chat, error) {
	if profileID <= 0 || profileUID == "" {
		return domain.Chat{}, ErrProfileNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	defer tx.Rollback(ctx)

	lockKey := fmt.Sprintf("private-chat:%d:%s", profileID, profileUID)
	if _, err = tx.Exec(ctx, acquirePrivateChatLockQuery, lockKey); err != nil {
		return domain.Chat{}, err
	}

	var profileCount int
	if err = tx.QueryRow(ctx, countActiveProfilesByUUIDForChatQuery, profileID, profileUID).Scan(&profileCount); err != nil {
		return domain.Chat{}, err
	}
	if profileCount != 2 {
		return domain.Chat{}, ErrProfileNotFound
	}

	var chatID int64
	err = tx.QueryRow(ctx, findPrivateChatByUUIDQuery, profileID, profileUID).Scan(&chatID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Chat{}, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(ctx, createChatQuery).Scan(&chatID); err != nil {
			return domain.Chat{}, err
		}
	}
	if _, err = tx.Exec(ctx, upsertParticipantsByUUIDQuery, profileID, chatID, profileUID); err != nil {
		return domain.Chat{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Chat{}, err
	}

	var chat domain.Chat
	var uid pgtype.UUID
	if err = r.pool.QueryRow(ctx, readChatQuery, chatID).Scan(&chat.ID, &uid, &chat.CreatedAt); err != nil {
		return domain.Chat{}, err
	}
	chat.UID = uuidString(uid)
	chat.Participants, err = r.readParticipants(ctx, chatID)
	if err != nil {
		return domain.Chat{}, err
	}
	return chat, nil
}

// ListChatsByUUID возвращает чаты с UUID-курсором без отдельного чтения ID курсора.
func (r *ChatRepository) ListChatsByUUID(ctx context.Context, profileID int64, cursorUID string, limit int) (domain.ChatsResponse, error) {
	limit = normalizeLimit(limit, 100)
	response := domain.ChatsResponse{Chats: make([]domain.Chat, 0), Limit: limit, CursorUID: cursorUID}
	rows, err := r.pool.Query(ctx, listChatsByUUIDQuery, profileID, cursorUID, limit+1)
	if err != nil {
		return response, err
	}
	for rows.Next() {
		var chat domain.Chat
		var uid pgtype.UUID
		if err := rows.Scan(&chat.ID, &uid, &chat.CreatedAt); err != nil {
			rows.Close()
			return response, err
		}
		chat.UID = uuidString(uid)
		response.Chats = append(response.Chats, chat)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return response, err
	}
	rows.Close()
	response.HasMore = len(response.Chats) > limit
	if response.HasMore {
		response.Chats = response.Chats[:limit]
	}
	if len(response.Chats) > 0 {
		response.NextCursorUID = response.Chats[len(response.Chats)-1].UID
	}
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
		if chat.LastMessage != nil {
			chat.LastMessage.ChatUID = chat.UID
		}
	}
	return response, nil
}

// IsParticipantByUUID проверяет участие пользователя в чате по UUID чата.
func (r *ChatRepository) IsParticipantByUUID(ctx context.Context, chatUID string, profileID int64) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, checkParticipantByUUIDQuery, chatUID, profileID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotChatParticipant
	}
	return nil
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
		var uid pgtype.UUID
		if err = rows.Scan(&chat.ID, &uid, &chat.CreatedAt); err != nil {
			rows.Close()
			return response, err
		}
		chat.UID = uuidString(uid)
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
		if chat.LastMessage != nil {
			chat.LastMessage.ChatUID = chat.UID
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
		var avatarURL pgtype.Text
		profile, _, scanErr := scanProfileWithLeadingAvatar(rows, &chatID, &avatarURL)
		if scanErr != nil {
			return nil, scanErr
		}
		if avatarURL.Valid {
			profile.AvatarURLs = []string{avatarURL.String}
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
