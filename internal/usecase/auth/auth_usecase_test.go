package auth

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"kvmm/internal/domain"
	authmock "kvmm/internal/mocks/auth"
	"kvmm/internal/repository"
)

func TestRegisterLoginCurrentAndLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := authmock.NewMockprofileRepository(ctrl)
	sessions := authmock.NewMocksessionRepository(ctrl)
	useCase := New(profiles, sessions)
	profile := domain.Profile{ID: 1, Nickname: "alice"}

	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	profiles.EXPECT().Create(gomock.Any(), gomock.Any()).Return(profile, nil)
	sessions.EXPECT().Create(int64(1)).Return("session", "csrf", nil)
	response, token, err := useCase.Register(context.Background(), domain.RegisterRequest{Nickname: "alice", Password: "password1"})
	if err != nil || response.Profile.ID != 1 || token != "session" {
		t.Fatalf("Register() = %+v, %q, %v", response, token, err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.MinCost)
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{Profile: profile, PasswordHash: string(hash)}, nil)
	sessions.EXPECT().Create(int64(1)).Return("session", "csrf", nil)
	if _, token, err := useCase.Login(context.Background(), domain.LoginRequest{Login: "alice", Password: "password1"}); err != nil || token != "session" {
		t.Fatalf("Login() = %q, %v", token, err)
	}

	sessions.EXPECT().Read("session").Return(int64(1), "csrf", nil)
	profiles.EXPECT().ReadByID(gomock.Any(), int64(1)).Return(profile, nil)
	if me, err := useCase.Current(context.Background(), "session"); err != nil || me.Profile.ID != 1 {
		t.Fatalf("Current() = %+v, %v", me, err)
	}
	sessions.EXPECT().Read("session").Return(int64(1), "csrf", nil)
	sessions.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	sessions.EXPECT().Delete("session")
	if err := useCase.Logout("session", "csrf"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
}

func TestAuthErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := authmock.NewMockprofileRepository(ctrl)
	sessions := authmock.NewMocksessionRepository(ctrl)
	useCase := New(profiles, sessions)

	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{}, repository.ErrProfileNotFound)
	if _, _, err := useCase.Login(context.Background(), domain.LoginRequest{Login: "alice", Password: "password1"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(missing) error = %v", err)
	}
	profiles.EXPECT().ReadByLoginAuth(gomock.Any(), "alice").Return(repository.AuthProfile{}, errors.New("database unavailable"))
	if _, _, err := useCase.Register(context.Background(), domain.RegisterRequest{Nickname: "alice", Password: "password1"}); err == nil {
		t.Fatal("Register() expected repository error")
	}
	sessions.EXPECT().Read("missing").Return(int64(0), "", repository.ErrSessionNotFound)
	if _, err := useCase.Current(context.Background(), "missing"); err == nil {
		t.Fatal("Current() expected session error")
	}
}
