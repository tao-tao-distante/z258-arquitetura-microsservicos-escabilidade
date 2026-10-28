package handler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
)

type fakeService struct {
	user  *domain.User
	token string
	err   error
}

func (f *fakeService) Register(ctx context.Context, email, password string) (*domain.User, string, error) {
	return f.user, f.token, f.err
}

func (f *fakeService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	return f.user, f.token, f.err
}

func TestRegisterSuccess(t *testing.T) {
	h := New(&fakeService{user: &domain.User{ID: 1, Email: "a@b.com"}, token: "tok"}, nil)
	body := bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", body))
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"token":"tok"`)
}

func TestRegisterInvalidJSON(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterMissingFields(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"a@b.com"}`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterEmailTaken(t *testing.T) {
	h := New(&fakeService{err: domain.ErrEmailTaken}, nil)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)))
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestRegisterInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRegisterInvalidInput(t *testing.T) {
	h := New(&fakeService{err: domain.ErrInvalidInput}, nil)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"bad","password":"password"}`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginSuccess(t *testing.T) {
	h := New(&fakeService{user: &domain.User{ID: 1, Email: "a@b.com"}, token: "tok"}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"token":"tok"`)
}

func TestLoginInvalidJSON(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginMissingFields(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com"}`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLoginInvalidCredentials(t *testing.T) {
	h := New(&fakeService{err: domain.ErrInvalidCredentials}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLoginInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"pw"}`)))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestLoginInvalidInput(t *testing.T) {
	h := New(&fakeService{err: domain.ErrInvalidInput}, nil)
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"bad","password":"password"}`)))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHealthOK(t *testing.T) {
	h := New(&fakeService{}, map[string]func(context.Context) error{
		"postgres": func(context.Context) error { return nil },
	})
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
	require.Contains(t, rec.Body.String(), `"postgres":"ok"`)
}

func TestHealthDegraded(t *testing.T) {
	h := New(&fakeService{}, map[string]func(context.Context) error{
		"postgres": func(context.Context) error { return errors.New("down") },
	})
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"degraded"`)
}

func TestRoutes(t *testing.T) {
	h := New(&fakeService{user: &domain.User{ID: 1, Email: "a@b.com"}, token: "tok"}, nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutesRejectsWrongMethods(t *testing.T) {
	h := New(&fakeService{}, nil)
	for _, path := range []string{"/register", "/login"} {
		rec := httptest.NewRecorder()
		h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestRegisterRejectsOversizedPayload(t *testing.T) {
	h := New(&fakeService{}, nil)
	body := append([]byte(`{"email":"a@b.com","password":"`), bytes.Repeat([]byte("x"), maxPayloadBytes)...)
	body = append(body, []byte(`"}`)...)
	rec := httptest.NewRecorder()
	h.Register(rec, httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body)))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestLoginRejectsTrailingJSON(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"email":"a@b.com","password":"password"}{}`)
	h.Login(rec, httptest.NewRequest(http.MethodPost, "/login", body))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInternalErrorLoggedOnceWithCorrelationID(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	h := New(&fakeService{err: errors.New("boom")}, nil, log)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"password"}`))
	req.Header.Set("x-correlation-id", "cid")
	h.Routes().ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, 1, bytes.Count(logs.Bytes(), []byte(`"msg":"internal_error"`)))
	require.Contains(t, logs.String(), `"correlation_id":"cid"`)
}
