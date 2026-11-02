package posts

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	postmock "kvmm/internal/mocks/post"
	sessionmock "kvmm/internal/mocks/session"
	postusecase "kvmm/internal/usecase/post"
	sessionusecase "kvmm/internal/usecase/session"
)

func postRequest(method, path string, value any) *http.Request {
	body, _ := json.Marshal(value)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	return request
}
func TestPostHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := postmock.NewMockpostRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	h := NewPostHandler(postusecase.New(posts, sessionusecase.New(session)))
	ctx := gomock.Any()
	authorID := int64(7)
	post := domain.Post{ID: 1, AuthorProfileID: &authorID}

	session.EXPECT().Read("session").Return(authorID, "csrf", nil).Times(3)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	posts.EXPECT().Create(ctx, gomock.Any()).Return(post, nil)
	posts.EXPECT().ReadByID(ctx, int64(1)).Return(post, nil).Times(2)
	posts.EXPECT().Update(ctx, int64(1), gomock.Any()).Return(post, nil)
	posts.EXPECT().Delete(ctx, int64(1)).Return(nil)
	posts.EXPECT().ReadProfilePosts(ctx, int64(7), int64(0), 10).Return([]domain.Post{post}, int64(2), true, nil)

	w := httptest.NewRecorder()
	h.Create(w, postRequest(http.MethodPost, "/api/posts/create", domain.CreatePostRequest{PostText: "hello"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("Create() status = %d", w.Code)
	}
	text := "updated"
	w = httptest.NewRecorder()
	h.Update(w, postRequest(http.MethodPatch, "/api/posts/update", domain.UpdatePostRequest{PostID: 1, PostText: &text}))
	if w.Code != http.StatusOK {
		t.Fatalf("Update() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Delete(w, postRequest(http.MethodDelete, "/api/posts/delete?post_id=1", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("Delete() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.UserPosts(w, httptest.NewRequest(http.MethodGet, "/api/posts/user?profile_id=7", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UserPosts() status = %d", w.Code)
	}
}

func TestPostHandlerValidationAndErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewPostHandler(postusecase.New(postmock.NewMockpostRepository(ctrl), sessionusecase.New(sessionmock.NewMockSessionReader(ctrl))))
	w := httptest.NewRecorder()
	h.Create(w, httptest.NewRequest(http.MethodGet, "/api/posts/create", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Delete(w, httptest.NewRequest(http.MethodDelete, "/api/posts/delete?post_id=bad", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.UserPosts(w, httptest.NewRequest(http.MethodGet, "/api/posts/user?profile_id=0", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid profile id status = %d", w.Code)
	}

	for _, test := range []struct {
		err    error
		status int
	}{
		{postusecase.ErrUnauthorized, http.StatusUnauthorized},
		{postusecase.ErrInvalidCSRF, http.StatusForbidden},
		{postusecase.ErrForbidden, http.StatusForbidden},
		{postusecase.ErrInvalidPost, http.StatusBadRequest},
		{postusecase.ErrPostNotFound, http.StatusNotFound},
		{errors.New("internal"), http.StatusInternalServerError},
	} {
		w = httptest.NewRecorder()
		h.writeError(w, test.err)
		if w.Code != test.status {
			t.Errorf("writeError(%v) status = %d", test.err, w.Code)
		}
	}
}

