package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	onlineconfv1 "kvmm/api/gen/onlineconf/v1"

	"kvmm/internal/domain"
	"kvmm/internal/handler/posts"
	"kvmm/internal/httpapi"
	postmock "kvmm/internal/mocks/post"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/onlineconfgrpc"
	"kvmm/internal/postsconfig"
	"kvmm/internal/repository"
	postusecase "kvmm/internal/usecase/post"
	sessionusecase "kvmm/internal/usecase/session"
)

type runtimePostRepository struct {
	*postmock.MockpostRepository
	likes int
}

func (r *runtimePostRepository) AttachLikeStats(context.Context, []domain.Post, int64) error {
	return nil
}

func (r *runtimePostRepository) LikeByUUID(context.Context, int64, string) error {
	r.likes++
	return nil
}

func (r *runtimePostRepository) UnlikeByUUID(context.Context, int64, string) error {
	r.likes--
	return nil
}

func (r *runtimePostRepository) ListPostFriendLikers(context.Context, int64, string, string, int) (domain.PostLikersResponse, error) {
	return domain.PostLikersResponse{}, nil
}

type runtimeSnapshotReader struct {
	current atomic.Pointer[domain.OnlineConfSnapshot]
}

func (r *runtimeSnapshotReader) Snapshot(_ context.Context, scope domain.OnlineConfScope) (domain.OnlineConfSnapshot, error) {
	if scope != domain.OnlineConfScopePosts {
		return domain.OnlineConfSnapshot{}, errors.New("unexpected snapshot scope")
	}
	current := r.current.Load()
	if current == nil {
		return domain.OnlineConfSnapshot{}, errors.New("snapshot unavailable")
	}
	return *current, nil
}

func TestPostsRuntimeIntegration(t *testing.T) {
	upstream := &runtimeSnapshotReader{}
	upstream.current.Store(&domain.OnlineConfSnapshot{
		Scope:   domain.OnlineConfScopePosts,
		Configs: map[string]json.RawMessage{"max_text_length": json.RawMessage("50")},
		Flags: map[string]domain.OnlineConfFlagSnapshot{
			"likes_enabled": {Rollout: 0, IncludeIDs: []int64{42}},
		},
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	const token = "test-token"
	server := grpc.NewServer()
	onlineconfv1.RegisterOnlineConfServiceServer(server, onlineconfgrpc.NewServer(upstream, token))
	go func() { _ = server.Serve(listener) }()
	defer listener.Close()
	defer server.Stop()

	client, err := onlineconfgrpc.Dial(listener.Addr().String(), token)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := postsconfig.New(client.Snapshot)
	ctrl := gomock.NewController(t)
	repo := &runtimePostRepository{MockpostRepository: postmock.NewMockpostRepository(ctrl)}
	sessions := sessionmock.NewMockSessionReader(ctrl)
	sessions.EXPECT().Read("test-session").Return(int64(42), "test-csrf", nil).AnyTimes()
	sessions.EXPECT().ValidateCSRF("test-session", "test-csrf").Return(nil).AnyTimes()
	useCase := postusecase.NewWithRuntime(repo, sessionusecase.New(sessions), cache, nil)
	router := httpapi.NewRouter(httpapi.Dependencies{Posts: posts.NewPostHandler(useCase)})

	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
		req.Header.Set("X-CSRF-Token", "test-csrf")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("%s %s returned %d, want %d: %s", method, path, w.Code, status, w.Body)
		}
		return w
	}
	assertRuntime := func(limit int, enabled bool) {
		t.Helper()
		w := request(http.MethodGet, "/api/posts/runtime", "", http.StatusOK)
		if header := w.Header().Get("Cache-Control"); header != "private, no-store" {
			t.Fatalf("Cache-Control = %q", header)
		}
		var values postsconfig.Values
		if err := json.Unmarshal(w.Body.Bytes(), &values); err != nil || values.MaxTextLength != limit || values.LikesEnabled != enabled {
			t.Fatalf("unexpected runtime response: %s", w.Body)
		}
	}
	assertCode := func(w *httptest.ResponseRecorder, code string) {
		t.Helper()
		var body domain.ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != code {
			t.Fatalf("expected error %q, got %s", code, w.Body)
		}
	}

	guest := httptest.NewRecorder()
	router.ServeHTTP(guest, httptest.NewRequest(http.MethodGet, "/api/posts/runtime", nil))
	if guest.Code != http.StatusUnauthorized || guest.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unexpected unauthenticated runtime response: %d %s", guest.Code, guest.Body)
	}

	assertRuntime(256, true)
	if err := cache.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	assertRuntime(50, true)

	const uid = "550e8400-e29b-41d4-a716-446655440000"
	const likePath = "/api/posts/" + uid + "/like"
	exact := strings.Repeat("🙂", 50)
	over := exact + "🙂"
	profileID := int64(42)
	repo.EXPECT().Create(gomock.Any(), repository.CreatePostParams{
		PostText: &exact, AuthorProfileID: &profileID,
	}).Return(domain.Post{}, nil)
	request(http.MethodPost, "/api/posts/create", `{"post_text":"  `+exact+`  "}`, http.StatusCreated)
	assertCode(request(http.MethodPost, "/api/posts/create", `{"post_text":"`+over+`"}`, http.StatusBadRequest), "post_text_too_long")
	assertCode(request(http.MethodPatch, "/api/posts/update", `{"post_uid":"`+uid+`","post_text":"`+over+`"}`, http.StatusBadRequest), "post_text_too_long")
	request(http.MethodPut, likePath, "", http.StatusNoContent)

	upstream.current.Store(&domain.OnlineConfSnapshot{
		Scope:   domain.OnlineConfScopePosts,
		Configs: map[string]json.RawMessage{"max_text_length": json.RawMessage("60")},
		Flags: map[string]domain.OnlineConfFlagSnapshot{
			"likes_enabled": {Rollout: 100, ExcludeIDs: []int64{42}},
		},
	})
	pollCtx, stopPoll := context.WithCancel(ctx)
	pollDone := make(chan struct{})
	go func() {
		defer close(pollDone)
		cache.Poll(pollCtx, func(err error) { t.Error(err) })
	}()
	defer func() {
		stopPoll()
		<-pollDone
	}()
	deadline := time.Now().Add(20 * time.Second)
	for cache.Values(42).LikesEnabled && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	assertRuntime(60, false)
	assertCode(request(http.MethodPut, likePath, "", http.StatusForbidden), "feature_disabled")
	assertCode(request(http.MethodDelete, likePath, "", http.StatusForbidden), "feature_disabled")
	if repo.likes != 1 {
		t.Fatalf("like repository mutations = %d", repo.likes)
	}

	stopPoll()
	<-pollDone

	upstream.current.Store(nil)
	if err := cache.Refresh(ctx); err == nil {
		t.Fatal("expected refresh to fail")
	}
	assertRuntime(60, false)

	upstream.current.Store(&domain.OnlineConfSnapshot{Scope: domain.OnlineConfScopePosts})
	if err := cache.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	assertRuntime(256, true)
}
