package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"kvmm/internal/domain"
	authmock "kvmm/internal/mocks/auth"
	"kvmm/internal/repository"
	authusecase "kvmm/internal/usecase/auth"
)

func requestWithJSON(method, path string, value any) *http.Request {
	body, _ := json.Marshal(value)
	return httptest.NewRequest(method, path, bytes.NewReader(body))
}

func TestAuthHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := authmock.NewMockprofileRepository(ctrl)
	sessions := authmock.NewMocksessionRepository(ctrl)
	h := NewAuthHandler(authusecase.New(profiles, sessions))
	profile := domain.Profile{ID: 1, Nickname: "alice"}

	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(profile, nil)
	sessions.EXPECT().Create(int64(1)).Return("session", "csrf", nil)
	register := domain.RegisterRequest{Nickname: "alice", Password: "password1", ConfirmPassword: "password1", PrivacyPolicyAllowed: true}
	w := httptest.NewRecorder()
	h.Register(w, requestWithJSON(http.MethodPost, "/api/auth/register", register))
	if w.Code != http.StatusCreated {
		t.Fatalf("Register() status = %d", w.Code)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.MinCost)
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{Profile: profile, PasswordHash: string(hash)}, nil)
	sessions.EXPECT().Create(int64(1)).Return("session", "csrf", nil)
	w = httptest.NewRecorder()
	h.Login(w, requestWithJSON(http.MethodPost, "/api/auth/login", domain.LoginRequest{Login: "alice", Password: "password1"}))
	if w.Code != http.StatusOK {
		t.Fatalf("Login() status = %d", w.Code)
	}

	sessions.EXPECT().Read("session").Return(int64(1), "csrf", nil)
	profiles.EXPECT().ReadByID(gomock.Any(), int64(1)).Return(profile, nil)
	me := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	me.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	w = httptest.NewRecorder()
	h.Me(w, me)
	if w.Code != http.StatusOK {
		t.Fatalf("Me() status = %d", w.Code)
	}

	sessions.EXPECT().Read("session").Return(int64(1), "csrf", nil)
	sessions.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	sessions.EXPECT().Delete("session")
	logout := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	logout.Header.Set("X-CSRF-Token", "csrf")
	w = httptest.NewRecorder()
	h.Logout(w, logout)
	if w.Code != http.StatusNoContent {
		t.Fatalf("Logout() status = %d", w.Code)
	}
}

func TestAuthHandlerValidationAndMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewAuthHandler(authusecase.New(authmock.NewMockprofileRepository(ctrl), authmock.NewMocksessionRepository(ctrl)))
	w := httptest.NewRecorder()
	h.Login(w, httptest.NewRequest(http.MethodGet, "/api/auth/login", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Register(w, requestWithJSON(http.MethodPost, "/api/auth/register", map[string]any{"nickname": "x"}))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation status = %d", w.Code)
	}
}

func TestEmailHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := authmock.NewMockprofileRepository(ctrl)
	sessions := authmock.NewMocksessionRepository(ctrl)
	sender := authmock.NewMockemailSender(ctrl)
	h := NewAuthHandler(authusecase.New(profiles, sessions, sender))

	var registerCode string
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "register@example.com").Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	sender.EXPECT().SendAuthCode("register@example.com", gomock.Any()).DoAndReturn(func(_, code string) error {
		registerCode = code
		return nil
	})
	w := httptest.NewRecorder()
	h.RequestRegisterCode(w, requestWithJSON(http.MethodPost, "/api/auth/register/email/send/code", domain.RegisterEmailCodeRequest{
		Email: "register@example.com", PrivacyPolicyAllowed: true,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("RequestRegisterCode() status = %d", w.Code)
	}

	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(domain.Profile{ID: 3}, nil)
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), gomock.Any()).Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	sessions.EXPECT().Create(int64(3)).Return("register-session", "csrf", nil)
	w = httptest.NewRecorder()
	h.RegisterByEmail(w, requestWithJSON(http.MethodPost, "/api/auth/register/email/code", domain.RegisterByEmailRequest{
		Email: "register@example.com", Code: registerCode,
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("RegisterByEmail() status = %d", w.Code)
	}

	loginProfile := domain.Profile{ID: 4}
	var loginCode string
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "login@example.com").Return(repository.AuthProfile{Profile: loginProfile}, nil)
	sender.EXPECT().SendAuthCode("login@example.com", gomock.Any()).DoAndReturn(func(_, code string) error {
		loginCode = code
		return nil
	})
	w = httptest.NewRecorder()
	h.RequestLoginCode(w, requestWithJSON(http.MethodPost, "/api/auth/login/email/send/code", domain.EmailCodeRequest{Email: "login@example.com"}))
	if w.Code != http.StatusOK {
		t.Fatalf("RequestLoginCode() status = %d", w.Code)
	}

	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "login@example.com").Return(repository.AuthProfile{Profile: loginProfile}, nil)
	sessions.EXPECT().Create(int64(4)).Return("login-session", "csrf", nil)
	w = httptest.NewRecorder()
	h.LoginByEmail(w, requestWithJSON(http.MethodPost, "/api/auth/login/email/code", domain.LoginByEmailRequest{
		Email: "login@example.com", Code: loginCode,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("LoginByEmail() status = %d", w.Code)
	}

	var changeCode string
	sessions.EXPECT().Read("session").Return(int64(4), "csrf", nil)
	sessions.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "new@example.com").Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	sender.EXPECT().SendAuthCode("new@example.com", gomock.Any()).DoAndReturn(func(_, code string) error {
		changeCode = code
		return nil
	})
	change := requestWithJSON(http.MethodPost, "/api/auth/email/change", domain.ChangeEmailRequest{Email: "new@example.com"})
	change.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	change.Header.Set("X-CSRF-Token", "csrf")
	w = httptest.NewRecorder()
	h.RequestEmailChange(w, change)
	if w.Code != http.StatusOK {
		t.Fatalf("RequestEmailChange() status = %d", w.Code)
	}

	sessions.EXPECT().Read("session").Return(int64(4), "csrf", nil)
	sessions.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	profiles.EXPECT().Update(gomock.Any(), int64(4), gomock.Any()).Return(domain.Profile{ID: 4}, nil)
	confirm := requestWithJSON(http.MethodPost, "/api/auth/email/confirm", domain.ConfirmEmailRequest{Email: "new@example.com", Code: changeCode})
	confirm.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	confirm.Header.Set("X-CSRF-Token", "csrf")
	w = httptest.NewRecorder()
	h.ConfirmEmail(w, confirm)
	if w.Code != http.StatusOK {
		t.Fatalf("ConfirmEmail() status = %d", w.Code)
	}
}

func TestEmailHandlersHideUnknownEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := authmock.NewMockprofileRepository(ctrl)
	sessions := authmock.NewMocksessionRepository(ctrl)
	sender := authmock.NewMockemailSender(ctrl)
	h := NewAuthHandler(authusecase.New(profiles, sessions, sender))

	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "unknown@example.com").Return(
		repository.AuthProfile{}, repository.ErrProfileNotFound,
	)
	sender.EXPECT().ReserveAuthCode("unknown@example.com").Return(true)

	w := httptest.NewRecorder()
	h.RequestLoginCode(w, requestWithJSON(http.MethodPost, "/api/auth/login/email/send/code", domain.EmailCodeRequest{
		Email: "unknown@example.com",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("RequestLoginCode(unknown email) status = %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"message":"verification code sent"`)) {
		t.Fatalf("RequestLoginCode(unknown email) body = %s", w.Body.String())
	}
}
