package service_test

import (
	"errors"
	"testing"

	"kvmm/internal/domain"
	"kvmm/internal/service"
	"kvmm/internal/store"
)

// TestRegister проверяет создание пользователя, сессии и CSRF-токена.
func TestRegister(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	profile, session, csrfToken, err := auth.Register(validRegisterRequest())
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if profile.ID == 0 || profile.Nickname != "vasily" {
		t.Errorf("unexpected profile: %+v", profile)
	}
	if session == "" || csrfToken == "" {
		t.Fatal("expected non-empty session and csrf token")
	}

	got, err := auth.ProfileBySession(session)
	if err != nil {
		t.Fatalf("get profile by session: %v", err)
	}
	if got.ID != profile.ID {
		t.Errorf("expected profile id %d, got %d", profile.ID, got.ID)
	}
}

// TestRegisterRejectsDuplicateContact проверяет уникальность nickname и контактов.
func TestRegisterRejectsDuplicateContact(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	if _, _, _, err := auth.Register(validRegisterRequest()); err != nil {
		t.Fatalf("register first user: %v", err)
	}

	duplicate := validRegisterRequest()
	duplicate.Nickname = "another"
	if _, _, _, err := auth.Register(duplicate); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict for duplicate contact, got %v", err)
	}
}

// TestLogin проверяет успешную авторизацию по nickname, email и телефону.
func TestLogin(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	if _, _, _, err := auth.Register(validRegisterRequest()); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logins := []string{"vasily", "vasily@example.com", "+79999999999"}
	for _, login := range logins {
		login := login
		t.Run(login, func(t *testing.T) {
			profile, session, csrfToken, err := auth.Login(domain.LoginRequest{
				Login:    login,
				Password: "qwerty123",
			})
			if err != nil {
				t.Fatalf("login: %v", err)
			}
			if profile.Nickname != "vasily" || session == "" || csrfToken == "" {
				t.Errorf("unexpected login result: profile=%+v session=%q csrf=%q", profile, session, csrfToken)
			}
		})
	}
}

// TestLoginRejectsInvalidCredentials проверяет отказ при неверном пароле и логине.
func TestLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	if _, _, _, err := auth.Register(validRegisterRequest()); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	tests := []domain.LoginRequest{
		{Login: "vasily", Password: "wrong-password"},
		{Login: "unknown", Password: "qwerty123"},
	}
	for _, request := range tests {
		if _, _, _, err := auth.Login(request); !errors.Is(err, service.ErrInvalidCredentials) {
			t.Errorf("expected invalid credentials for %+v, got %v", request, err)
		}
	}
}

// TestSessionAndCSRF проверяет получение CSRF-токена и его валидацию.
func TestSessionAndCSRF(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	_, session, csrfToken, err := auth.Register(validRegisterRequest())
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	gotToken, err := auth.CSRFTokenBySession(session)
	if err != nil {
		t.Fatalf("get csrf token: %v", err)
	}
	if gotToken != csrfToken {
		t.Errorf("expected csrf token %q, got %q", csrfToken, gotToken)
	}
	if err := auth.ValidateCSRF(session, csrfToken); err != nil {
		t.Fatalf("valid csrf token was rejected: %v", err)
	}
	if err := auth.ValidateCSRF(session, "wrong-token"); !errors.Is(err, service.ErrInvalidCSRF) {
		t.Fatalf("expected invalid csrf error, got %v", err)
	}
}

// TestLogout удаляет сессию и запрещает использовать её повторно.
func TestLogout(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(store.NewMemory())
	_, session, _, err := auth.Register(validRegisterRequest())
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	auth.Logout(session)
	if _, err := auth.ProfileBySession(session); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected deleted session, got %v", err)
	}
}

// TestFeedList проверяет offset, limit и признак наличия следующей страницы.
func TestFeedList(t *testing.T) {
	t.Parallel()

	feed := service.NewFeedService(store.NewMemory())
	posts, hasMore := feed.List(10, 5)
	if len(posts) != 5 {
		t.Fatalf("expected 5 posts, got %d", len(posts))
	}
	if !hasMore {
		t.Error("expected more posts after first page")
	}
	if posts[0].ID != 11 {
		t.Errorf("expected post id 11 at offset 10, got %d", posts[0].ID)
	}
}

// TestFeedListInvalidRange проверяет безопасное поведение ленты для некорректных параметров.
func TestFeedListInvalidRange(t *testing.T) {
	t.Parallel()

	feed := service.NewFeedService(store.NewMemory())
	for _, request := range [][2]int{{-1, 5}, {0, 0}, {100, 5}} {
		posts, hasMore := feed.List(request[0], request[1])
		if len(posts) != 0 || hasMore {
			t.Errorf("expected empty page for offset=%d limit=%d, got posts=%d hasMore=%t", request[0], request[1], len(posts), hasMore)
		}
	}
}

// validRegisterRequest возвращает корректные данные пользователя для тестов.
func validRegisterRequest() domain.RegisterRequest {
	return domain.RegisterRequest{
		Nickname:        "vasily",
		Email:           stringPointer("vasily@example.com"),
		PhoneNumber:     stringPointer("+79999999999"),
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
		ProfileName:     "Vasily",
		Surname:         "Demo",
		Gender:          domain.GenderMale,
	}
}

// stringPointer возвращает указатель на строковое значение для тестовых данных.
func stringPointer(value string) *string { return &value }
