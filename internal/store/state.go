package store

import (
	"embed"
	"errors"
	"sync"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/store/fake_data"
)

//go:embed media/*
var EmbeddedMediaFiles embed.FS

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidCSRF = errors.New("invalid csrf token")

// in-merory хранилище
var mu sync.RWMutex
var nextID int64 = 1
var users = map[int64]User{}
var sessions = map[string]Session{}
var posts = fake_data.MakePosts()

// Reset очищает in-memory данные (используется для тестов)
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	nextID = 1
	users = map[int64]User{}
	sessions = map[string]Session{}
}

// User хранит профиль и хеш пароля пользователя
type User struct {
	Profile      domain.Profile
	PasswordHash []byte
}

// Session хранит данные авторизованной сессии
type Session struct {
	UserID    int64
	CSRFToken string
}

// startPostGenerator добавляет новый пост каждую минуту
func StartPostGenerator() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		nextPostID := int64(101)

		for range ticker.C {
			post := fake_data.MakePost(int(nextPostID))

			mu.Lock()
			posts[nextPostID] = post
			mu.Unlock()

			nextPostID++
		}
	}()
}
