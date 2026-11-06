package feed

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	feedmock "kvmm/internal/mocks/feed"
	feedusecase "kvmm/internal/usecase/feed"
)

func TestFeedHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := feedmock.NewMockpostRepository(ctrl)
	posts.EXPECT().ReadFeedPosts(gomock.Any(), int64(1), 1).Return([]domain.Post{{ID: 1}}, int64(2), true, nil)
	h := NewFeedHandler(feedusecase.New(posts))
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/feed?cursor=1&limit=1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("List() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/feed?limit=0", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodPost, "/api/feed", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("invalid method status = %d", w.Code)
	}
}

func TestFeedHandlerInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := feedmock.NewMockpostRepository(ctrl)
	posts.EXPECT().ReadFeedPosts(gomock.Any(), int64(0), 10).Return(nil, int64(0), false, errors.New("database unavailable"))
	w := httptest.NewRecorder()
	NewFeedHandler(feedusecase.New(posts)).List(w, httptest.NewRequest(http.MethodGet, "/api/feed", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal error status = %d", w.Code)
	}
}
