package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"kvmm/internal/domain"
	"kvmm/internal/handler/auth"
	"kvmm/internal/handler/feed"
	"kvmm/internal/store"
)

const sessionCookieName = "session_id"

// TestRegister проверяет регистрацию пользователя, JSON-ответ и cookie сессии
func TestRegister(t *testing.T) {
	store.Reset()

	h := auth.NewAuthHandler()
	body := bytes.NewBufferString(`{
		"nickname":"vasily",
		"password":"qwerty123",
		"confirm_password":"qwerty123",
		"privacy_policy_allowed":true
	}`)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	w := httptest.NewRecorder()

	h.Register(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response domain.AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Profile.Nickname != "vasily" {
		t.Errorf("expected nickname %q, got %q", "vasily", response.Profile.Nickname)
	}
	if response.CSRFToken == "" {
		t.Error("expected csrf token in response")
	}
	if cookie := w.Result().Cookies(); len(cookie) != 1 || cookie[0].Name != sessionCookieName {
		t.Fatalf("expected session cookie, got %v", cookie)
	}
}

// TestLogin проверяет авторизацию ранее зарегистрированного пользователя
func TestLogin(t *testing.T) {
	store.Reset()

	if _, _, _, err := store.Register(domain.RegisterRequest{
		Nickname:        "vasily",
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := auth.NewAuthHandler()
	body := bytes.NewBufferString(`{"login":"vasily","password":"qwerty123"}`)

	r := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var response domain.AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Profile.Nickname != "vasily" || response.CSRFToken == "" {
		t.Errorf("unexpected login response: %+v", response)
	}
}

// TestRegisterEmailCodeRejectsExistingEmail проверяет запрет повторной регистрации по email.
func TestRegisterEmailCodeRejectsExistingEmail(t *testing.T) {
	store.Reset()

	code, err := store.CreateEmailCode("ivan@example.com", store.EmailCodeRegister)
	if err != nil {
		t.Fatalf("create registration code: %v", err)
	}
	if _, _, _, err := store.RegisterByEmail(domain.RegisterByEmailRequest{
		Email: "ivan@example.com",
		Code:  code,
	}); err != nil {
		t.Fatalf("register by email: %v", err)
	}

	body := bytes.NewBufferString(`{"email":"ivan@example.com","privacy_policy_allowed":true}`)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/register/email/code", body)
	w := httptest.NewRecorder()

	auth.NewAuthHandler().RequestRegisterCode(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, w.Code, w.Body.String())
	}
}

// TestMe проверяет получение текущего пользователя по cookie сессии
func TestMe(t *testing.T) {
	store.Reset()

	profile, session, csrfToken, err := store.Register(domain.RegisterRequest{
		Nickname:        "vasily",
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := auth.NewAuthHandler()

	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session})
	w := httptest.NewRecorder()

	h.Me(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var response domain.MeResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	expected := domain.MeResponse{Profile: profile, CSRFToken: csrfToken}
	if !reflect.DeepEqual(response, expected) {
		t.Errorf("expected %+v, got %+v", expected, response)
	}
}

// TestLogoutChecksCSRF проверяет обязательную проверку CSRF-токена при выходе
func TestLogoutChecksCSRF(t *testing.T) {
	store.Reset()

	_, session, csrfToken, err := store.Register(domain.RegisterRequest{
		Nickname:        "vasily",
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := auth.NewAuthHandler()

	withoutCSRF := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	withoutCSRF.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session})
	w := httptest.NewRecorder()
	h.Logout(w, withoutCSRF)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d without csrf, got %d", http.StatusForbidden, w.Code)
	}

	withCSRF := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	withCSRF.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session})
	withCSRF.Header.Set("X-CSRF-Token", csrfToken)
	w = httptest.NewRecorder()
	h.Logout(w, withCSRF)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d with csrf, got %d", http.StatusNoContent, w.Code)
	}
}

// TestFeedList проверяет выдачу страницы ленты с cursor и limit
func TestFeedList(t *testing.T) {
	store.Reset()

	feed := feed.NewFeedHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/posts?cursor=10&limit=5", nil)
	w := httptest.NewRecorder()

	feed.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	var response domain.FeedResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Posts) != 5 || response.Cursor != 10 || response.Limit != 5 {
		t.Errorf("unexpected feed response: %+v", response)
	}
}

// TestMethodNotAllowed проверяет ответ для неподдерживаемого HTTP-метода
func TestMethodNotAllowed(t *testing.T) {
	store.Reset()

	h := auth.NewAuthHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
	if got := w.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("expected Allow header %q, got %q", http.MethodPost, got)
	}
}

// TestRegisterValidation проверяет минимальную валидацию регистрации.
func TestRegisterValidation(t *testing.T) {
	store.Reset()

	tests := []struct {
		name  string
		setup func(*domain.RegisterRequest)
	}{
		{name: "nickname is required", setup: func(req *domain.RegisterRequest) { req.Nickname = "" }},
		{name: "nickname is too short", setup: func(req *domain.RegisterRequest) { req.Nickname = "abc" }},
		{name: "password is too short", setup: func(req *domain.RegisterRequest) { req.Password = "short" }},
		{name: "password confirmation does not match", setup: func(req *domain.RegisterRequest) { req.ConfirmPassword = "different123" }},
		{name: "nickname has special characters", setup: func(req *domain.RegisterRequest) { req.Nickname = "vasi<ly>" }},
		{name: "password is 72 cyrillic letters", setup: withPassword(strings.Repeat("я", 72))},
		{name: "password has cyrillic letters", setup: withPassword("пароль123")},
		{name: "password has space", setup: withPassword("qwerty 123")},
		{name: "password has no digits", setup: withPassword("qwertyuiop")},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			store.Reset()

			req := validRegisterRequest()
			test.setup(&req)
			body, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			h := auth.NewAuthHandler()
			r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.Register(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d, response: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

// TestRegisterAcceptsValidValues проверяет допустимые nickname и пароль.
func TestRegisterAcceptsValidValues(t *testing.T) {
	tests := []string{"vasily", "ivan_67", "anna.demo"}

	for _, nickname := range tests {
		t.Run(nickname, func(t *testing.T) {
			store.Reset()

			req := validRegisterRequest()
			req.Nickname = nickname
			body, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()
			auth.NewAuthHandler().Register(w, r)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected status %d, got %d, response: %s", http.StatusCreated, w.Code, w.Body.String())
			}
		})
	}
}

// TestLoginValidation проверяет, что логин и пароль вне правил отклоняются до сравнения с хешем
func TestLoginValidation(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
	}{
		{name: "login is too long", login: strings.Repeat("a", 255), password: "qwerty123"},
		{name: "password is too long", login: "vasily", password: strings.Repeat("a", 64) + "1"},
		{name: "password has cyrillic letters", login: "vasily", password: "пароль123"},
		{name: "password has zero bytes", login: "vasily", password: "qwerty123\x00qwerty123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.Reset()

			body, err := json.Marshal(domain.LoginRequest{Login: test.login, Password: test.password})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			w := httptest.NewRecorder()
			auth.NewAuthHandler().Login(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d, response: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

// TestRequestBodyLimit проверяет, что слишком большое тело запроса отклоняется
func TestRequestBodyLimit(t *testing.T) {
	store.Reset()

	body := `{"nickname":"` + strings.Repeat("a", 1<<20) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	auth.NewAuthHandler().Register(w, r)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "too large") {
		t.Fatalf("expected body limit error, got %d: %s", w.Code, w.Body.String())
	}
}

// withPassword возвращает настройку запроса с одинаковыми паролем и подтверждением
func withPassword(password string) func(*domain.RegisterRequest) {
	return func(req *domain.RegisterRequest) { req.Password = password; req.ConfirmPassword = password }
}

// validRegisterRequest возвращает корректные данные регистрации для тестов
func validRegisterRequest() domain.RegisterRequest {
	return domain.RegisterRequest{
		Nickname:             "vasily",
		Password:             "qwerty123",
		ConfirmPassword:      "qwerty123",
		PrivacyPolicyAllowed: true,
	}
}
