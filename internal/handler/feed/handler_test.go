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
	posts.EXPECT().ReadFeedPostsByUUID(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000", 1).Return([]domain.Post{{ID: 1, UID: "550e8400-e29b-41d4-a716-446655440000"}}, "660e8400-e29b-41d4-a716-446655440000", true, nil)
	h := NewFeedHandler(feedusecase.New(posts))
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/feed?cursor=550e8400-e29b-41d4-a716-446655440000&limit=1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("List() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.List(w, httptest.NewRequest(http.MethodGet, "/api/feed?limit=0", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status = %d", w.Code)
	}
}

func TestFeedHandlerInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	posts := feedmock.NewMockpostRepository(ctrl)
	posts.EXPECT().ReadFeedPostsByUUID(gomock.Any(), "", 10).Return(nil, "", false, errors.New("database unavailable"))
	w := httptest.NewRecorder()
	NewFeedHandler(feedusecase.New(posts)).List(w, httptest.NewRequest(http.MethodGet, "/api/feed", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal error status = %d", w.Code)
	}
}
