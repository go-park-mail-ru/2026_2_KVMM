package helpers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"kvmm/internal/domain"
)

func TestPaginationAndPositiveQuery(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?cursor=2&limit=5", nil)
	cursor, limit, err := Pagination(r, 10)
	if err != nil || cursor != 2 || limit != 5 {
		t.Fatalf("Pagination() = %d, %d, %v", cursor, limit, err)
	}
	for _, query := range []string{"cursor=-1", "cursor=bad", "limit=0", "limit=11"} {
		if _, _, err := Pagination(httptest.NewRequest(http.MethodGet, "/?"+query, nil), 10); err == nil {
			t.Fatalf("Pagination(%q) expected error", query)
		}
	}
	r = httptest.NewRequest(http.MethodGet, "/?id=7", nil)
	if id, err := PositiveQueryInt64(r, "id"); err != nil || id != 7 {
		t.Fatalf("PositiveQueryInt64() = %d, %v", id, err)
	}
	if _, err := PositiveQueryInt64(httptest.NewRequest(http.MethodGet, "/?id=0", nil), "id"); err == nil {
		t.Fatal("PositiveQueryInt64() expected error")
	}
}

func TestMethodAndCookies(t *testing.T) {
	w := httptest.NewRecorder()
	if MethodAllowed(w, httptest.NewRequest(http.MethodGet, "/", nil), http.MethodPost) {
		t.Fatal("MethodAllowed() returned true for wrong method")
	}
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	SetSessionCookie(w, "session")
	if len(w.Result().Cookies()) != 1 {
		t.Fatal("session cookie was not set")
	}
	w = httptest.NewRecorder()
	DeleteSessionCookie(w)
	if w.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatal("session cookie was not deleted")
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session"})
	if SessionCookie(r) != "session" {
		t.Fatal("SessionCookie() returned wrong value")
	}
}

func TestJSONHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, "bad", "bad request")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"code":"bad"`) {
		t.Fatalf("WriteError() = %d, %s", w.Code, w.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"ok"}`))
	var value struct {
		Value string `json:"value"`
	}
	w = httptest.NewRecorder()
	if err := DecodeJSON(w, request, &value); err != nil || value.Value != "ok" {
		t.Fatalf("DecodeJSON() = %+v, %v", value, err)
	}
	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unknown":true}`))
	w = httptest.NewRecorder()
	if err := DecodeJSON(w, request, &value); err == nil {
		t.Fatal("DecodeJSON() expected unknown field error")
	}
}

func TestValidators(t *testing.T) {
	valid := domain.RegisterRequest{Nickname: "alice_1", Password: "password1", ConfirmPassword: "password1", PrivacyPolicyAllowed: true}
	if err := ValidateRegisterRequest(valid); err != nil {
		t.Fatalf("ValidateRegisterRequest(valid) = %v", err)
	}
	for _, request := range []domain.RegisterRequest{
		{Nickname: "1alice", Password: "password1"},
		{Nickname: "alice", Password: "password"},
	} {
		if err := ValidateRegisterRequest(request); err == nil {
			t.Fatal("ValidateRegisterRequest(invalid) expected error")
		}
	}
	name := "Alice"
	password := "password1"
	confirm := "password1"
	if err := ValidateUpdateProfileRequest(domain.UpdateProfileRequest{ProfileName: &name, Password: &password, ConfirmPassword: &confirm}); err != nil {
		t.Fatalf("ValidateUpdateProfileRequest(valid) = %v", err)
	}
	if err := ValidateUpdateProfileRequest(domain.UpdateProfileRequest{}); err == nil {
		t.Fatal("empty profile update expected error")
	}
	badEmail := "wrong"
	if err := ValidateUpdateProfileRequest(domain.UpdateProfileRequest{Email: &badEmail}); err == nil {
		t.Fatal("invalid email expected error")
	}
	badPassword := "password"
	if err := ValidateUpdateProfileRequest(domain.UpdateProfileRequest{Password: &badPassword, ConfirmPassword: &confirm}); err == nil {
		t.Fatal("invalid password expected error")
	}
}

func TestInternalErrorAndSameSite(t *testing.T) {
	w := httptest.NewRecorder()
	WriteInternalError(w, "test", os.ErrClosed)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "internal_error") {
		t.Fatalf("WriteInternalError() = %d, %s", w.Code, w.Body.String())
	}
	for _, value := range []string{"strict", "none", "lax"} {
		t.Setenv("COOKIE_SAMESITE", value)
		w = httptest.NewRecorder()
		SetSessionCookie(w, "session")
		if len(w.Result().Cookies()) != 1 {
			t.Fatalf("cookie was not set for SameSite=%s", value)
		}
	}
}
