package session

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	sessionmock "kvmm/internal/mocks/session"
)

func TestReadAndAuthorize(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := sessionmock.NewMockSessionReader(ctrl)
	useCase := New(reader)

	if _, _, err := useCase.Read(""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Read(empty) error = %v", err)
	}
	reader.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	if userID, csrf, err := useCase.Read("session"); err != nil || userID != 7 || csrf != "csrf" {
		t.Fatalf("Read() = %d, %q, %v", userID, csrf, err)
	}
	reader.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	reader.EXPECT().ValidateCSRF("session", "csrf").Return(nil)
	if _, err := useCase.Authorize("session", "csrf"); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	reader.EXPECT().Read("session").Return(int64(7), "csrf", nil)
	reader.EXPECT().ValidateCSRF("session", "wrong").Return(errors.New("bad csrf"))
	if _, err := useCase.Authorize("session", "wrong"); !errors.Is(err, ErrInvalidCSRF) {
		t.Fatalf("Authorize(invalid csrf) error = %v", err)
	}
}

func TestReadStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := sessionmock.NewMockSessionReader(ctrl)
	reader.EXPECT().Read("session").Return(int64(0), "", errors.New("storage unavailable"))
	if _, _, err := New(reader).Read("session"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Read() error = %v", err)
	}
}
