package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"kvmm/internal/domain"
	"kvmm/internal/handler"
	"kvmm/internal/store"
)

const sessionCookieName = "session_id"

// TestRegister проверяет регистрацию пользователя, JSON-ответ и cookie сессии
func TestRegister(t *testing.T) {
	store.Reset()

	h := handler.NewAuthHandler()
	body := bytes.NewBufferString(`{
		"nickname":"vasily",
		"email":"vasily@example.com",
		"password":"qwerty123",
		"confirm_password":"qwerty123",
		"profile_name":"Vasily",
		"surname":"Demo",
		"gender":"male",
		"birthday":"2000-05-14"
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
		Email:           stringPointer("vasily@example.com"),
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
		ProfileName:     "Vasily",
		Surname:         "Demo",
		Gender:          domain.GenderMale,
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := handler.NewAuthHandler()
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

// TestMe проверяет получение текущего пользователя по cookie сессии
func TestMe(t *testing.T) {
	store.Reset()

	profile, session, csrfToken, err := store.Register(domain.RegisterRequest{
		Nickname:        "vasily",
		Email:           stringPointer("vasily@example.com"),
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
		ProfileName:     "Vasily",
		Surname:         "Demo",
		Gender:          domain.GenderMale,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := handler.NewAuthHandler()

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
		Email:           stringPointer("vasily@example.com"),
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
		ProfileName:     "Vasily",
		Surname:         "Demo",
		Gender:          domain.GenderMale,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	h := handler.NewAuthHandler()

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

	feed := handler.NewFeedHandler()
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

	h := handler.NewAuthHandler()
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

// TestRegisterValidation проверяет валидацию всех основных полей регистрации
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
		{name: "email has invalid format", setup: func(req *domain.RegisterRequest) { req.Email = stringPointer("invalid-email") }},
		{name: "phone has invalid format", setup: func(req *domain.RegisterRequest) { req.PhoneNumber = stringPointer("+0") }},
		{name: "profile name has digits", setup: func(req *domain.RegisterRequest) { req.ProfileName = "Ivan123" }},
		{name: "surname has special characters", setup: func(req *domain.RegisterRequest) { req.Surname = "Ivan!" }},
		{name: "patronymic has special characters", setup: func(req *domain.RegisterRequest) { value := "Ivan@"; req.Patronymic = &value }},
		{name: "gender is invalid", setup: func(req *domain.RegisterRequest) { req.Gender = "unknown" }},
		{name: "birthday has invalid format", setup: func(req *domain.RegisterRequest) { value := "14.05.2000"; req.Birthday = &value }},
		{name: "bio is too long", setup: func(req *domain.RegisterRequest) { value := strings.Repeat("a", 257); req.Bio = &value }},
		{name: "contact is required", setup: func(req *domain.RegisterRequest) { req.Email = nil; req.PhoneNumber = nil }},
		{name: "nickname has special characters", setup: func(req *domain.RegisterRequest) { req.Nickname = "vasi<ly>" }},
		{name: "email has special characters", setup: func(req *domain.RegisterRequest) { req.Email = stringPointer("va!sily@example.com") }},
		{name: "phone is too short", setup: func(req *domain.RegisterRequest) { req.PhoneNumber = stringPointer("+7999") }},
		{name: "profile name has double space", setup: func(req *domain.RegisterRequest) { req.ProfileName = "Mary  Ann" }},
		{name: "gender other is not allowed", setup: func(req *domain.RegisterRequest) { req.Gender = domain.GenderOther }},
		{name: "password is 72 cyrillic letters", setup: withPassword(strings.Repeat("я", 72))},
		{name: "password has cyrillic letters", setup: withPassword("пароль123")},
		{name: "password has space", setup: withPassword("qwerty 123")},
		{name: "password has no digits", setup: withPassword("qwertyuiop")},
		{name: "birthday is required", setup: func(req *domain.RegisterRequest) { req.Birthday = nil }},
		{name: "age is under 14", setup: func(req *domain.RegisterRequest) { req.Birthday = birthdayForAge(14, 1) }},
		{name: "age is over 120", setup: func(req *domain.RegisterRequest) { req.Birthday = birthdayForAge(121, 0) }},
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

			h := handler.NewAuthHandler()
			r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.Register(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d, response: %s", http.StatusBadRequest, w.Code, w.Body.String())
			}
		})
	}
}

// TestRegisterAcceptsValidValues проверяет, что допустимые имена, контакты и граничный возраст проходят валидацию
func TestRegisterAcceptsValidValues(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*domain.RegisterRequest)
	}{
		{name: "profile name with space", setup: func(req *domain.RegisterRequest) { req.ProfileName = "Mary Ann" }},
		{name: "surname with hyphen", setup: func(req *domain.RegisterRequest) { req.Surname = "Римский-Корсаков" }},
		{name: "surname with apostrophe", setup: func(req *domain.RegisterRequest) { req.Surname = "O'Neil" }},
		{name: "phone without email", setup: func(req *domain.RegisterRequest) { req.Email = nil }},
		{name: "age is 14", setup: func(req *domain.RegisterRequest) { req.Birthday = birthdayForAge(14, 0) }},
		{name: "age is 120", setup: func(req *domain.RegisterRequest) { req.Birthday = birthdayForAge(121, 1) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.Reset()

			req := validRegisterRequest()
			test.setup(&req)
			body, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handler.NewAuthHandler().Register(w, r)

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
			handler.NewAuthHandler().Login(w, r)

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
	handler.NewAuthHandler().Register(w, r)

	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "too large") {
		t.Fatalf("expected body limit error, got %d: %s", w.Code, w.Body.String())
	}
}

// stringPointer возвращает указатель на строковое значение для тестовых данных
func stringPointer(value string) *string { return &value }

// withPassword возвращает настройку запроса с одинаковыми паролем и подтверждением
func withPassword(password string) func(*domain.RegisterRequest) {
	return func(req *domain.RegisterRequest) { req.Password = password; req.ConfirmPassword = password }
}

// birthdayForAge возвращает дату рождения, при которой years лет исполнится через days дней
func birthdayForAge(years, days int) *string {
	return stringPointer(time.Now().UTC().AddDate(-years, 0, days).Format(time.DateOnly))
}

// validRegisterRequest возвращает корректные данные регистрации для тестов
func validRegisterRequest() domain.RegisterRequest {
	return domain.RegisterRequest{
		Nickname:        "vasily",
		Email:           stringPointer("vasily@example.com"),
		PhoneNumber:     stringPointer("+79999999999"),
		Password:        "qwerty123",
		ConfirmPassword: "qwerty123",
		ProfileName:     "Vasily",
		Surname:         "Demo",
		Patronymic:      stringPointer("Demoovich"),
		Gender:          domain.GenderMale,
		Birthday:        stringPointer("2000-05-14"),
		Bio:             stringPointer("hi!"),
	}
}
