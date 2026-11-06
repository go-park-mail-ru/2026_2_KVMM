package relationship

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"kvmm/internal/domain"
	relationshipmock "kvmm/internal/mocks/relationship"
	sessionmock "kvmm/internal/mocks/session"
	"kvmm/internal/repository"
	relusecase "kvmm/internal/usecase/relationship"
	sessionusecase "kvmm/internal/usecase/session"
)

func relationshipRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	request.Header.Set("X-CSRF-Token", "csrf")
	return request
}

func relationshipGet(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})
	return request
}

func TestRelationshipHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	relationships := relationshipmock.NewMockrelationshipRepository(ctrl)
	session := sessionmock.NewMockSessionReader(ctrl)
	h := NewRelationshipHandler(relusecase.New(relationships, sessionusecase.New(session)))
	ctx := gomock.Any()

	session.EXPECT().Read("session").Return(int64(7), "csrf", nil).Times(7)
	session.EXPECT().ValidateCSRF("session", "csrf").Return(nil).Times(3)
	relationships.EXPECT().List(ctx, int64(7), gomock.Any()).Return(domain.RelationshipsResponse{}, nil)
	relationships.EXPECT().Types(ctx, int64(7)).Return(domain.RelationshipTypes{}, nil)
	relationships.EXPECT().Requests(ctx, int64(7), "", int64(0), 10).Return(domain.RelationshipsResponse{}, nil)
	relationships.EXPECT().Suggestions(ctx, int64(7), int64(0), 10).Return(domain.RelationshipSuggestionsResponse{}, nil)
	relationships.EXPECT().Create(ctx, int64(7), gomock.Any()).Return(domain.ProfileRelationship{}, nil)
	relationships.EXPECT().Accept(ctx, int64(7), gomock.Any()).Return(domain.ProfileRelationship{}, nil)
	relationships.EXPECT().Cancel(ctx, int64(7), gomock.Any()).Return(domain.ProfileRelationship{}, nil)

	w := httptest.NewRecorder()
	h.List(w, relationshipGet("/api/relationships/list?limit=1"))
	if w.Code != http.StatusOK {
		t.Fatalf("List() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.TypesList(w, relationshipGet("/api/relationships/typeslist"))
	if w.Code != http.StatusOK {
		t.Fatalf("TypesList() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Requests(w, relationshipGet("/api/relationships/requests"))
	if w.Code != http.StatusOK {
		t.Fatalf("Requests() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Suggestions(w, relationshipGet("/api/relationships/suggestions"))
	if w.Code != http.StatusOK {
		t.Fatalf("Suggestions() status = %d", w.Code)
	}
	body := `{"profile_id_slave":2,"relationships_type":"friendship"}`
	w = httptest.NewRecorder()
	h.Create(w, relationshipRequest(http.MethodPost, "/api/relationships/create", body))
	if w.Code != http.StatusCreated {
		t.Fatalf("Create() status = %d", w.Code)
	}
	body = `{"profile_id":2,"relationships_type":"friendship"}`
	w = httptest.NewRecorder()
	h.Accept(w, relationshipRequest(http.MethodPatch, "/api/relationships/accept", body))
	if w.Code != http.StatusOK {
		t.Fatalf("Accept() status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.Cancle(w, relationshipRequest(http.MethodDelete, "/api/relationships/cancle", body))
	if w.Code != http.StatusOK {
		t.Fatalf("Cancle() status = %d", w.Code)
	}
}

func TestRelationshipHandlerValidationAndErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	h := NewRelationshipHandler(relusecase.New(relationshipmock.NewMockrelationshipRepository(ctrl), sessionusecase.New(sessionmock.NewMockSessionReader(ctrl))))
	w := httptest.NewRecorder()
	h.Create(w, relationshipRequest(http.MethodPost, "/api/relationships/create", `{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation status = %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.List(w, relationshipGet("/api/relationships/list?limit=0"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("pagination status = %d", w.Code)
	}
	for _, test := range []struct {
		err    error
		status int
	}{
		{relusecase.ErrUnauthorized, http.StatusUnauthorized},
		{relusecase.ErrInvalidCSRF, http.StatusForbidden},
		{relusecase.ErrConflict, http.StatusConflict},
		{relusecase.ErrNotFound, http.StatusNotFound},
		{relusecase.ErrInvalid, http.StatusBadRequest},
		{repository.ErrRelationshipNotFound, http.StatusNotFound},
		{errors.New("internal"), http.StatusInternalServerError},
	} {
		w = httptest.NewRecorder()
		h.writeError(w, test.err)
		if w.Code != test.status {
			t.Errorf("writeError(%v) status = %d", test.err, w.Code)
		}
	}
}
