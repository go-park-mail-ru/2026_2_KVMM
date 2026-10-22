package repository

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
)

var (
	ErrSessionNotFound = errors.New("session not found")  // сессия не существует
	ErrInvalidCSRF     = errors.New("invalid csrf token") // CSRF-токен не совпал с токеном сессии
)

type session struct {
	userID    int64
	csrfToken string
	// TODO: feat: Добавить гео данаые сессии через GeoLite2
}

// SessionRepository хранит активные сессии приложения
type SessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]session
}

// NewSessionRepository создаёт хранилище сессий в памяти процесса
func NewSessionRepository() *SessionRepository {
	return &SessionRepository{sessions: make(map[string]session)}
}

// Create создаёт сессию и возвращает session cookie и CSRF-токен
func (r *SessionRepository) Create(userID int64) (string, string, error) {
	sessionToken, err := randomToken()
	if err != nil {
		return "", "", err
	}
	csrfToken, err := randomToken()
	if err != nil {
		return "", "", err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions == nil {
		r.sessions = make(map[string]session)
	}
	r.sessions[sessionToken] = session{userID: userID, csrfToken: csrfToken}
	return sessionToken, csrfToken, nil
}

// Read возвращает пользователя и CSRF-токен по ID
func (r *SessionRepository) Read(token string) (int64, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	current, ok := r.sessions[token]
	if !ok || token == "" {
		return 0, "", ErrSessionNotFound
	}
	return current.userID, current.csrfToken, nil
}

// ValidateCSRF проверяет CSRF-токен для сессии
func (r *SessionRepository) ValidateCSRF(token, csrfToken string) error {
	_, expected, err := r.Read(token)
	if err != nil || csrfToken == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(csrfToken)) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

// Delete завершает сессию
func (r *SessionRepository) Delete(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, token)
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
