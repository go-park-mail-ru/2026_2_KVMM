package profile

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	profilemock "kvmm/internal/mocks/profile"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	profileusecase "kvmm/internal/usecase/profile"
	sessionusecase "kvmm/internal/usecase/session"
)

func profileRequest(method, path string, body *bytes.Buffer) *http.Request {
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, body)
	}
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	return request
}

func TestProfileHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := profilemock.NewMockprofileRepository(ctrl)
	avatars := profilemock.NewMockavatarRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	h := NewProfileHandler(profileusecase.New(profiles, avatars, sessionusecase.New(session)))
	ctx := gomock.Any()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).Times(4)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	profiles.EXPECT().Update(ctx, int64(7), gomock.Any()).Return(domain.Profile{ID: 7}, nil)
	avatars.EXPECT().List(ctx, int64(7)).Return(domain.AvatarsResponse{Avatars: []domain.Avatar{{ID: 1}}}, nil)
	avatars.EXPECT().Delete(ctx, int64(7), int64(1)).Return(nil)
	avatars.EXPECT().Create(ctx, int64(7), gomock.Any()).Return(domain.Avatar{ID: 1}, nil)

	w := httptest.NewRecorder()
	h.Update(w, profileRequest(http.MethodPatch, "/api/profile/update", bytes.NewBufferString(`{"profile_name":"Alice"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("Update() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ListAvatars(w, profileRequest(http.MethodGet, "/api/profile/avatars", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAvatars() status = %d", w.Code)
	}
	request := profileRequest(http.MethodDelete, "/api/profile/avatars/1", nil)
	request.SetPathValue("id", "1")
	w = httptest.NewRecorder()
	h.DeleteAvatar(w, request)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteAvatar() status = %d", w.Code)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("image"))
	_ = writer.Close()
	request = profileRequest(http.MethodPost, "/api/profile/avatar", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	h.UploadAvatar(w, request)
	if w.Code != http.StatusCreated {
		t.Fatalf("UploadAvatar() status = %d", w.Code)
	}
}

func TestProfileSearchHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	profiles := profilemock.NewMockprofileRepository(ctrl)
	avatars := profilemock.NewMockavatarRepository(ctrl)
	h := NewProfileHandler(profileusecase.New(
		profiles,
		avatars,
		sessionusecase.New(sessionmock.NewMockSessionReader(ctrl)),
	))

	profiles.EXPECT().ReadByLogin(gomock.Any(), "alice").Return(domain.Profile{
		ID:       1,
		Nickname: "alice",
	}, nil)

	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/profile/search?login=alice", nil)
	h.Search(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("Search() status = %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `"nickname":"alice"`) {
		t.Fatalf("Search() body = %s", body)
	}

	profiles.EXPECT().ReadByID(gomock.Any(), int64(1)).Return(domain.Profile{
		ID:       1,
		Nickname: "alice",
	}, nil)
	w = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/profile/search?profile_id=1", nil)
	h.Search(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("Search(profile_id) status = %d", w.Code)
	}
}

func TestProfileHandlerValidationAndErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewProfileHandler(profileusecase.New(profilemock.NewMockprofileRepository(ctrl), profilemock.NewMockavatarRepository(ctrl), sessionusecase.New(sessionmock.NewMockSessionReader(ctrl))))
	w := httptest.NewRecorder()
	h.Update(w, profileRequest(http.MethodPatch, "/api/profile/update", bytes.NewBufferString(`{}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty update status = %d", w.Code)
	}
	request := profileRequest(http.MethodDelete, "/api/profile/avatars/bad", nil)
	request.SetPathValue("id", "bad")
	w = httptest.NewRecorder()
	h.DeleteAvatar(w, request)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid avatar id status = %d", w.Code)
	}
	for _, test := range []struct {
		err    error
		status int
	}{
		{profileusecase.ErrUnauthorized, http.StatusUnauthorized},
		{profileusecase.ErrInvalidCSRF, http.StatusForbidden},
		{profileusecase.ErrAvatarLimit, http.StatusConflict},
		{profileusecase.ErrAvatarNotFound, http.StatusNotFound},
		{profileusecase.ErrInvalidAvatar, http.StatusBadRequest},
		{repository.ErrProfileConflict, http.StatusInternalServerError},
	} {
		w = httptest.NewRecorder()
		h.writeAvatarError(w, test.err)
		if w.Code != test.status {
			t.Errorf("writeAvatarError(%v) status = %d", test.err, w.Code)
		}
	}
}
