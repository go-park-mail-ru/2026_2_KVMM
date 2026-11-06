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
