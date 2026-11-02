package handler

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
)

func TestAuthBusinessLogsUseLevelsAndHidePII(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))

	registered := New(&fakeService{user: &domain.User{ID: 7, Email: "ana@example.com", Name: "Ana"}, token: "tok"}, nil, log, &fakeCounter{}, testSecret)
	rec := httptest.NewRecorder()
	registered.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"ana@example.com","password":"secret123"}`)))
	require.Equal(t, http.StatusCreated, rec.Code)

	loggedIn := New(&fakeService{user: &domain.User{ID: 7, Email: "ana@example.com", Name: "Ana"}, token: "tok"}, nil, log, &fakeCounter{}, testSecret)
	rec = httptest.NewRecorder()
	loggedIn.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"ana@example.com","password":"secret123"}`)))
	require.Equal(t, http.StatusOK, rec.Code)

	rejected := New(&fakeService{err: domain.ErrInvalidCredentials}, nil, log, &fakeCounter{}, testSecret)
	rec = httptest.NewRecorder()
	rejected.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"ana@example.com","password":"secret123"}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	profile := New(&fakeService{user: &domain.User{ID: 7, Email: "ana@example.com", Name: "Beatriz"}}, nil, log, &fakeCounter{}, testSecret)
	require.Equal(t, http.StatusOK, serveProfile(t, profile, http.MethodPatch, "/me", `{"name":"Beatriz"}`, true).Code)

	password := New(&fakeService{}, nil, log, &fakeCounter{}, testSecret)
	require.Equal(t, http.StatusNoContent, serveProfile(t, password, http.MethodPut, "/me/password", `{"current_password":"secret123","new_password":"new-secret123"}`, true).Code)

	output := logs.String()
	for _, event := range []string{"user_registered", "login_succeeded", "login_failed", "profile_updated", "password_changed"} {
		require.Contains(t, output, `"msg":"`+event+`"`)
	}
	require.Contains(t, output, `"level":"WARN"`)
	require.Contains(t, output, `"user_id":7`)
	require.Contains(t, output, `"reason":"invalid_credentials"`)
	require.NotContains(t, output, "ana@example.com")
	require.NotContains(t, output, "Beatriz")
	require.NotContains(t, output, "secret123")
	require.NotContains(t, output, "new-secret123")
}

func TestRegisterInternalErrorStillLogsAtErrorLevel(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	h := New(&fakeService{err: errors.New("boom")}, nil, log, &fakeCounter{}, testSecret)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"ana@example.com","password":"secret123"}`)))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, logs.String(), `"level":"ERROR"`)
	require.Contains(t, logs.String(), `"msg":"internal_error"`)
}
