package handler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
)

const testSecret = "test-secret"

func (f *fakeService) Profile(ctx context.Context, userID int64) (*domain.User, error) {
	f.gotUserID = userID
	return f.user, f.err
}

func (f *fakeService) UpdateName(ctx context.Context, userID int64, name string) (*domain.User, error) {
	f.gotUserID = userID
	f.gotName = name
	return f.user, f.err
}

func (f *fakeService) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	f.gotUserID = userID
	f.gotCurrent, f.gotNext = current, next
	return f.err
}

func bearer(t *testing.T, userID int64) string {
	t.Helper()
	token, err := auth.GenerateToken(testSecret, userID, "a@b.com", time.Hour)
	require.NoError(t, err)
	return "Bearer " + token
}

func serveProfile(t *testing.T, h *Handler, method, path, body string, authorized bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if authorized {
		req.Header.Set("Authorization", bearer(t, 42))
	}
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name       string
		svc        *fakeService
		authorized bool
		wantCode   int
		wantBody   string
		wantLabel  []string
	}{
		{name: "success", svc: &fakeService{user: &domain.User{ID: 42, Email: "a@b.com", Name: "Ana"}}, authorized: true, wantCode: http.StatusOK, wantBody: `{"id":42,"email":"a@b.com","name":"Ana"}`, wantLabel: []string{"get_profile", "success"}},
		{name: "missing token", svc: &fakeService{}, wantCode: http.StatusUnauthorized, wantLabel: []string{"get_profile", "unauthorized"}},
		{name: "not found", svc: &fakeService{err: domain.ErrNotFound}, authorized: true, wantCode: http.StatusNotFound, wantLabel: []string{"get_profile", "not_found"}},
		{name: "internal error", svc: &fakeService{err: errors.New("boom")}, authorized: true, wantCode: http.StatusInternalServerError, wantLabel: []string{"get_profile", "error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			h := New(tt.svc, nil, slog.Default(), counter, testSecret)
			rec := serveProfile(t, h, http.MethodGet, "/me", "", tt.authorized)
			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantBody != "" {
				require.JSONEq(t, tt.wantBody, rec.Body.String())
				require.Equal(t, int64(42), tt.svc.gotUserID)
			}
			require.Equal(t, [][]string{tt.wantLabel}, counter.labels)
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name      string
		svc       *fakeService
		body      string
		wantCode  int
		wantLabel []string
	}{
		{name: "success", svc: &fakeService{user: &domain.User{ID: 42, Email: "a@b.com", Name: "Bia"}}, body: `{"name":"Bia"}`, wantCode: http.StatusOK, wantLabel: []string{"update_profile", "success"}},
		{name: "invalid json", svc: &fakeService{}, body: `{`, wantCode: http.StatusBadRequest, wantLabel: []string{"update_profile", "invalid_payload"}},
		{name: "unknown field", svc: &fakeService{}, body: `{"name":"Bia","email":"x@y.com"}`, wantCode: http.StatusBadRequest, wantLabel: []string{"update_profile", "invalid_payload"}},
		{name: "invalid name", svc: &fakeService{err: domain.ErrInvalidInput}, body: `{"name":""}`, wantCode: http.StatusBadRequest, wantLabel: []string{"update_profile", "invalid_input"}},
		{name: "not found", svc: &fakeService{err: domain.ErrNotFound}, body: `{"name":"Bia"}`, wantCode: http.StatusNotFound, wantLabel: []string{"update_profile", "not_found"}},
		{name: "internal error", svc: &fakeService{err: errors.New("boom")}, body: `{"name":"Bia"}`, wantCode: http.StatusInternalServerError, wantLabel: []string{"update_profile", "error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			h := New(tt.svc, nil, slog.Default(), counter, testSecret)
			rec := serveProfile(t, h, http.MethodPatch, "/me", tt.body, true)
			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCode == http.StatusOK {
				require.JSONEq(t, `{"id":42,"email":"a@b.com","name":"Bia"}`, rec.Body.String())
				require.Equal(t, "Bia", tt.svc.gotName)
			}
			require.Equal(t, [][]string{tt.wantLabel}, counter.labels)
		})
	}
}

func TestUpdateProfileRejectsOversizedPayload(t *testing.T) {
	h := newTestHandler(&fakeService{}, nil)
	body := `{"name":"` + string(bytes.Repeat([]byte("x"), maxPayloadBytes)) + `"}`
	rec := serveProfile(t, h, http.MethodPatch, "/me", body, true)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestChangePassword(t *testing.T) {
	const valid = `{"current_password":"password","new_password":"new-password"}`
	tests := []struct {
		name      string
		svc       *fakeService
		body      string
		wantCode  int
		wantLabel []string
	}{
		{name: "success", svc: &fakeService{}, body: valid, wantCode: http.StatusNoContent, wantLabel: []string{"change_password", "success"}},
		{name: "invalid json", svc: &fakeService{}, body: `{`, wantCode: http.StatusBadRequest, wantLabel: []string{"change_password", "invalid_payload"}},
		{name: "missing fields", svc: &fakeService{}, body: `{"current_password":"password"}`, wantCode: http.StatusBadRequest, wantLabel: []string{"change_password", "invalid_payload"}},
		{name: "invalid new password", svc: &fakeService{err: domain.ErrInvalidInput}, body: valid, wantCode: http.StatusBadRequest, wantLabel: []string{"change_password", "invalid_input"}},
		{name: "wrong current password", svc: &fakeService{err: domain.ErrWrongPassword}, body: valid, wantCode: http.StatusForbidden, wantLabel: []string{"change_password", "wrong_password"}},
		{name: "not found", svc: &fakeService{err: domain.ErrNotFound}, body: valid, wantCode: http.StatusNotFound, wantLabel: []string{"change_password", "not_found"}},
		{name: "internal error", svc: &fakeService{err: errors.New("boom")}, body: valid, wantCode: http.StatusInternalServerError, wantLabel: []string{"change_password", "error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			h := New(tt.svc, nil, slog.Default(), counter, testSecret)
			rec := serveProfile(t, h, http.MethodPut, "/me/password", tt.body, true)
			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCode == http.StatusNoContent {
				require.Empty(t, rec.Body.String())
				require.Equal(t, int64(42), tt.svc.gotUserID)
				require.Equal(t, "password", tt.svc.gotCurrent)
				require.Equal(t, "new-password", tt.svc.gotNext)
			}
			require.Equal(t, [][]string{tt.wantLabel}, counter.labels)
		})
	}
}

func TestChangePasswordIsRateLimited(t *testing.T) {
	counter := &fakeCounter{}
	h := New(&fakeService{}, nil, slog.Default(), counter, testSecret)
	var rec *httptest.ResponseRecorder
	for range 11 {
		req := httptest.NewRequest(http.MethodPut, "/me/password", bytes.NewBufferString(`{"current_password":"password","new_password":"new-password"}`))
		req.Header.Set("Authorization", bearer(t, 42))
		req.RemoteAddr = "192.0.2.20:1234"
		rec = httptest.NewRecorder()
		h.Routes().ServeHTTP(rec, req)
	}
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, []string{"change_password", "rate_limited"}, counter.labels[len(counter.labels)-1])
}

func TestProfileRoutesFailClosedWithoutSecret(t *testing.T) {
	h := New(&fakeService{user: &domain.User{ID: 42}}, nil, slog.Default(), &fakeCounter{}, "")
	rec := serveProfile(t, h, http.MethodGet, "/me", "", true)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProfileHandlersRejectMissingClaims(t *testing.T) {
	h := newTestHandler(&fakeService{user: &domain.User{ID: 42}}, nil)
	for _, handle := range []http.HandlerFunc{h.GetProfile, h.UpdateProfile, h.ChangePassword} {
		rec := httptest.NewRecorder()
		handle(rec, httptest.NewRequest(http.MethodGet, "/me", bytes.NewBufferString(`{}`)))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
}

func TestLoginResponseIncludesName(t *testing.T) {
	h := newTestHandler(&fakeService{user: &domain.User{ID: 1, Email: "a@b.com", Name: "Ana"}, token: "tok"}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"password"}`)))
	require.JSONEq(t, `{"id":1,"email":"a@b.com","name":"Ana","token":"tok"}`, rec.Body.String())
}
