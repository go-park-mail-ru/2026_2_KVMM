package profile

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	profilemock "kvmm/internal/mocks/profile"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	sessionusecase "kvmm/internal/usecase/session"
)

func TestProfileUseCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := profilemock.NewMockprofileRepository(ctrl)
	avatars := profilemock.NewMockavatarRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(profiles, avatars, sessionusecase.New(session))
	ctx := context.Background()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).Times(4)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	profiles.EXPECT().Update(ctx, int64(7), gomock.Any()).Return(domain.Profile{ID: 7}, nil)
	avatars.EXPECT().Create(ctx, int64(7), gomock.Any()).Return(domain.Avatar{ID: 1}, nil)
	avatars.EXPECT().List(ctx, int64(7)).Return(domain.AvatarsResponse{Avatars: []domain.Avatar{{ID: 1}}}, nil)
	avatars.EXPECT().Delete(ctx, int64(7), int64(1)).Return(nil)

	name := "Alice"
	if _, err := useCase.Update(ctx, "session", "csrf", domain.UpdateProfileRequest{ProfileName: &name}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if _, err := useCase.UploadAvatar(ctx, "session", "csrf", domain.UploadAvatarRequest{Data: []byte("image")}); err != nil {
		t.Fatalf("UploadAvatar() error = %v", err)
	}
	if _, err := useCase.ListAvatars(ctx, "session"); err != nil {
		t.Fatalf("ListAvatars() error = %v", err)
	}
	if err := useCase.DeleteAvatar(ctx, "session", "csrf", 1); err != nil {
		t.Fatalf("DeleteAvatar() error = %v", err)
	}
}

func TestProfileUseCaseErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := profilemock.NewMockprofileRepository(ctrl)
	avatars := profilemock.NewMockavatarRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(profiles, avatars, sessionusecase.New(session))
	ctx := context.Background()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	badBirthday := "not-a-date"
	if _, err := useCase.Update(ctx, "session", "csrf", domain.UpdateProfileRequest{Birthday: &badBirthday}); err == nil {
		t.Fatal("Update() expected birthday error")
	}
	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	profiles.EXPECT().Update(ctx, int64(7), gomock.Any()).Return(domain.Profile{}, repository.ErrProfileConflict)
	if _, err := useCase.Update(ctx, "session", "csrf", domain.UpdateProfileRequest{}); !errors.Is(err, repository.ErrProfileConflict) {
		t.Fatalf("Update(conflict) error = %v", err)
	}
	session.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	avatars.EXPECT().Create(ctx, int64(7), gomock.Any()).Return(domain.Avatar{}, repository.ErrAvatarLimit)
	if _, err := useCase.UploadAvatar(ctx, "session", "csrf", domain.UploadAvatarRequest{}); !errors.Is(err, repository.ErrAvatarLimit) {
		t.Fatalf("UploadAvatar(limit) error = %v", err)
	}
}
