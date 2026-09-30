package store

import (
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"kvmm/internal/domain"
)

//go:embed media/*
var embeddedMediaFiles embed.FS

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type User struct {
	Profile      domain.Profile
	PasswordHash []byte
}

type session struct {
	userID    int64
	csrfToken string
}

// Memory хранит пользователей, сессии и публикации в памяти процесса.
type Memory struct {
	mu       sync.RWMutex
	nextID   int64
	users    map[int64]User
	sessions map[string]session
	posts    map[int64]domain.Post
}

// NewMemory создаёт in-memory хранилище с тестовыми пользователями и постами.
func NewMemory() *Memory {
	createdAt := time.Now().UTC()
	firstAvatar := "/api/media/avatar1.png"
	secondAvatar := "/api/media/avatar2.png"
	firstText := `Что такое Остров Херон

Это крошечный кусочек суши посреди океана, в южной части Большого Барьерного рифа. Размер — всего около 800 на 300 метров. То есть ты можешь обойти его пешком минут за двадцать.

Остров полностью занят одним курортом и исследовательской станцией Университета Квинсленда. Никаких других отелей, никаких магазинов, никаких «городских» развлечений там нет.

Как туда попасть

Сначала летишь из Брисбена в городок Гладстон — это где-то час самолетом. Потом от пристани Гладстона идешь на катере примерно два часа. Катер ходит один раз в день в каждую сторону. Или можно долететь вертолетом, если хочется побыстрее и подороже.

Почему туда едут

Главная причина — черепахи. С ноября по март на остров приплывают зеленые и логгерхедовые черепахи, чтобы откладывать яйца. Прямо на пляже, куда ты можешь выйти в любой момент. А с января по май из песка начинают вылупляться маленькие черепашки и бегут к воде.

Второй момент — риф прямо у берега. Не надо плыть на лодке, не надо никуда ехать. Просто заходишь в воду и плывешь. Под тобой кораллы, рыбы, иногда небольшие акулы и скаты. Вода прозрачная, особенно в зимние месяцы.`

	secondText := `Здесь время течет иначе. Оно пахнет специями на рынках Марракеша, звучит криками муэдзинов на рассвете и теряется в бесконечных улочках синей медины Шефшауэна.

Марокко бьет по всем органам чувств сразу:
👁 Глаза: Рыжие стены ancient kasbah, бирюзовые двери и закаты над Сахарой, которые окрашивают песок в цвет золота.
👂 Уши: Ритмы гнауа, шум торга и полная, звенящая тишина в пустыне.
👃 Нос: Аромат мяты, кедра, кожи и таджина с курицей и лимоном.

Мы пили чай с берберами, спали под звездами в пустыне Эрг-Шебби и учились торговаться так, чтобы остаться в плюсе (спойлер: не всегда получалось 😂).

Марокко нельзя понять умом. Ее нужно просто впустить в сердце. Кто был — поймет. Кто не был — обязательно добавьте в свой вишлист!`

	m := &Memory{
		nextID:   101,
		users:    map[int64]User{},
		sessions: map[string]session{},
		posts:    make(map[int64]domain.Post, 100),
	}

	for i := 1; i <= 100; i++ {
		id := int64(i)
		avatar := firstAvatar
		gender := domain.GenderFemale
		text := firstText
		mediaURLs := []string{
			"/api/media/post_media1.jpg",
			"/api/media/post_media1.pdf",
			"/api/media/post_media1.pdf.zip",
		}
		likes := int64(4)
		comments := int64(1)
		reposts := int64(2)

		if i%2 == 0 {
			avatar = secondAvatar
			gender = domain.GenderMale
			text = secondText
			mediaURLs = []string{"/api/media/post_media2.1.png", "/api/media/post_media2.2.mp4"}
			likes = 7
			comments = 13
		}

		profile := domain.Profile{
			ID:          id,
			Nickname:    fmt.Sprintf("demo%d", i),
			ProfileName: "Демо",
			Surname:     "Пользователь",
			Gender:      gender,
			ImageURL:    avatar,
			AvatarURLs:  []string{avatar},
			CreatedAt:   createdAt,
		}
		m.posts[id] = domain.Post{
			ID:              id,
			PostText:        &text,
			AuthorProfileID: &profile.ID,
			Author:          profile,
			CreatedAt:       createdAt.Add(-time.Duration(i) * time.Minute),
			MediaCount:      int64(len(mediaURLs)),
			MediaURLs:       mediaURLs,
			LikesCount:      likes,
			CommentsCount:   comments,
			RepostsCount:    reposts,
		}
	}
	return m
}

// ReadMediaFile возвращает встроенный файл публикации по безопасному имени.
func ReadMediaFile(name string) ([]byte, error) {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) {
		return nil, ErrNotFound
	}
	return embeddedMediaFiles.ReadFile("media/" + name)
}

// CreateUser сохраняет нового пользователя после проверки уникальности.
func (m *Memory) CreateUser(user User) (domain.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.users {
		if strings.EqualFold(existing.Profile.Nickname, user.Profile.Nickname) ||
			(existing.Profile.Email != nil && user.Profile.Email != nil && strings.EqualFold(*existing.Profile.Email, *user.Profile.Email)) ||
			(existing.Profile.PhoneNumber != nil && user.Profile.PhoneNumber != nil && *existing.Profile.PhoneNumber == *user.Profile.PhoneNumber) {
			return domain.Profile{}, ErrConflict
		}
	}
	user.Profile.ID = m.nextID
	m.nextID++
	m.users[user.Profile.ID] = user
	return user.Profile, nil
}

// FindUser ищет пользователя по nickname, email или номеру телефона.
func (m *Memory) FindUser(login string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if strings.EqualFold(user.Profile.Nickname, login) ||
			(user.Profile.Email != nil && strings.EqualFold(*user.Profile.Email, login)) ||
			(user.Profile.PhoneNumber != nil && *user.Profile.PhoneNumber == login) {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

// UserByID возвращает пользователя по числовому идентификатору.
func (m *Memory) UserByID(id int64) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

// SaveSession сохраняет сессию пользователя и CSRF-токен.
func (m *Memory) SaveSession(token string, userID int64, csrfToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = session{userID: userID, csrfToken: csrfToken}
}

// SessionUserID возвращает идентификатор пользователя по сессии.
func (m *Memory) SessionUserID(token string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[token]
	if !ok {
		return 0, ErrNotFound
	}
	return session.userID, nil
}

// SessionCSRFToken возвращает CSRF-токен сессии.
func (m *Memory) SessionCSRFToken(token string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[token]
	if !ok {
		return "", ErrNotFound
	}
	return session.csrfToken, nil
}

// DeleteSession удаляет сессию пользователя.
func (m *Memory) DeleteSession(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// Feed возвращает часть ленты от новых публикаций к старым.
func (m *Memory) Feed(offset, limit int) ([]domain.Post, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if offset < 0 || limit < 1 {
		return []domain.Post{}, false
	}
	result := make([]domain.Post, 0, len(m.posts))
	for _, post := range m.posts {
		result = append(result, post)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if offset >= len(result) {
		return []domain.Post{}, false
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], end < len(result)
}
