package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggerRecoversPanic(t *testing.T) {
	handler := requestLogger(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d", w.Code)
	}
}

func TestRequestLoggerAddsRequestID(t *testing.T) {
	handler := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loggingID := r.Context().Value(struct{}{}); loggingID != nil {
			t.Fatal("request ID leaked under an unrelated context key")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	request.Header.Set("X-Request-ID", "client-request-123")
	handler.ServeHTTP(w, request)
	if got := w.Header().Get("X-Request-ID"); got == "" || got == "client-request-123" {
		t.Fatalf("request ID header = %q", got)
	}
}

func TestRouterHealthAndCORS(t *testing.T) {
	router := NewRouter(Dependencies{})
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set("Origin", "http://localhost:8001")
	router.ServeHTTP(w, request)
	if w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8001" {
		t.Fatalf("health response = %d, headers = %v", w.Code, w.Header())
	}
	w = httptest.NewRecorder()
	options := httptest.NewRequest(http.MethodOptions, "/api/feed", nil)
	router.ServeHTTP(w, options)
	if w.Code != http.StatusNoContent {
		t.Fatalf("options status = %d", w.Code)
	}
}
