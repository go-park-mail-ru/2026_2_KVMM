package repository

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisSessionKeyPrefix = "kvmm:session:"

var (
	ErrSessionNotFound = errors.New("session not found")  // сессия не существует
	ErrInvalidCSRF     = errors.New("invalid csrf token") // CSRF-токен не совпал с токеном сессии
	ErrNoToken         = errors.New("please use session id")
)

// RedisSessionRepository хранит сессии в Redis
type RedisSessionRepository struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisSessionRepository создаёт Redis-хранилище сессий
func NewRedisSessionRepository(client *redis.Client, ttl time.Duration) (*RedisSessionRepository, error) {
	if client == nil {
		return nil, errors.New("redis client is nil")
	}
	if ttl <= 0 {
		return nil, errors.New("redis session TTL must be positive")
	}
	return &RedisSessionRepository{client: client, ttl: ttl}, nil
}

// Ping проверяет доступность Redis
func (r *RedisSessionRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Create создаёт сессию и возвращает session cookie и CSRF-токен
func (r *RedisSessionRepository) Create(userID int64) (string, string, error) {
	sessionToken, err := randomToken()
	if err != nil {
		return "", "", err
	}
	csrfToken, err := randomToken()
	if err != nil {
		return "", "", err
	}

	ctx := context.Background()
	key := redisSessionKey(sessionToken)
	if err := r.client.HSet(ctx, key, map[string]any{
		"user_id":    userID,
		"csrf_token": csrfToken,
	}).Err(); err != nil {
		return "", "", err
	}
	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		_ = r.client.Del(ctx, key).Err()
		return "", "", err
	}
	return sessionToken, csrfToken, nil
}

// Read возвращает пользователя и CSRF-токен по session cookie
func (r *RedisSessionRepository) Read(token string) (int64, string, error) {
	if strings.TrimSpace(token) == "" {
		return 0, "", ErrSessionNotFound
	}
	values, err := r.client.HGetAll(context.Background(), redisSessionKey(token)).Result()
	if err != nil {
		return 0, "", err
	}
	userID, err := strconv.ParseInt(values["user_id"], 10, 64)
	if len(values) == 0 || err != nil || userID <= 0 || values["csrf_token"] == "" {
		return 0, "", ErrSessionNotFound
	}
	return userID, values["csrf_token"], nil
}

// ValidateCSRF проверяет CSRF-токен без загрузки всей записи сессии
func (r *RedisSessionRepository) ValidateCSRF(token, csrfToken string) error {
	if strings.TrimSpace(token) == "" || csrfToken == "" {
		return ErrInvalidCSRF
	}
	expected, err := r.client.HGet(context.Background(), redisSessionKey(token), "csrf_token").Result()
	if err != nil || subtle.ConstantTimeCompare([]byte(expected), []byte(csrfToken)) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

// Delete завершает сессию
func (r *RedisSessionRepository) Delete(token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrNoToken
	}
	if err := r.client.Del(context.Background(), redisSessionKey(token)).Err(); err != nil {
		return ErrNoToken
	}
	return nil
}

func redisSessionKey(token string) string {
	return redisSessionKeyPrefix + token
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
