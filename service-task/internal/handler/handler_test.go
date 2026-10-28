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

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/auth"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

type fakeService struct {
	task   *domain.Task
	tasks  []domain.Task
	err    error
	limit  int
	offset int
}

func (f *fakeService) Create(ctx context.Context, userID int64, title, description string) (*domain.Task, error) {
	return f.task, f.err
}

func (f *fakeService) List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error) {
	f.limit, f.offset = limit, offset
	return f.tasks, f.err
}

func (f *fakeService) Get(ctx context.Context, id, userID int64) (*domain.Task, error) {
	return f.task, f.err
}

func (f *fakeService) Update(ctx context.Context, id, userID int64, title, description, status string) (*domain.Task, error) {
	return f.task, f.err
}

func (f *fakeService) Delete(ctx context.Context, id, userID int64) error {
	return f.err
}

func token(t *testing.T, secret string) string {
	t.Helper()
	claims := auth.Claims{UserID: 1, Email: "a@b.com", RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func ctxWithClaims() context.Context {
	return auth.WithClaims(context.Background(), &auth.Claims{UserID: 1})
}

func TestListSuccess(t *testing.T) {
	svc := &fakeService{tasks: []domain.Task{{ID: 1}}}
	h := New(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil).WithContext(ctxWithClaims())
	rec := httptest.NewRecorder()
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"id":1`)
	require.Equal(t, 50, svc.limit)
	require.Zero(t, svc.offset)
}

func TestListAcceptsOptionalPagination(t *testing.T) {
	svc := &fakeService{}
	h := New(svc, nil)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/tasks?limit=20&offset=40", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 20, svc.limit)
	require.Equal(t, 40, svc.offset)
}

func TestListRejectsUnsafePagination(t *testing.T) {
	h := New(&fakeService{}, nil)
	for _, query := range []string{"limit=0", "limit=101", "limit=x", "offset=-1"} {
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequest(http.MethodGet, "/tasks?"+query, nil).WithContext(ctxWithClaims()))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func TestCreateRejectsOversizedPayload(t *testing.T) {
	h := New(&fakeService{}, nil)
	body := bytes.NewBufferString(`{"title":"x","description":"` + string(bytes.Repeat([]byte("x"), maxPayloadBytes)) + `"}`)
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", body).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestListError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestListUnauthorized(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateSuccess(t *testing.T) {
	h := New(&fakeService{task: &domain.Task{ID: 1, Title: "x"}}, nil)
	body := bytes.NewBufferString(`{"title":"x","description":"d"}`)
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", body).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreateInvalidJSON(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateInvalidInput(t *testing.T) {
	h := New(&fakeService{err: domain.ErrInvalidInput}, nil)
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":""}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"x"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetSuccess(t *testing.T) {
	h := New(&fakeService{task: &domain.Task{ID: 1}}, nil)
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestGetInvalidID(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/tasks/abc", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetNotFound(t *testing.T) {
	h := New(&fakeService{err: domain.ErrNotFound}, nil)
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUpdateSuccess(t *testing.T) {
	h := New(&fakeService{task: &domain.Task{ID: 1}}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{"title":"x","status":"done"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateInvalidJSON(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateInvalidID(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/abc", bytes.NewBufferString(`{}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateInvalidInput(t *testing.T) {
	h := New(&fakeService{err: domain.ErrInvalidInput}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{"title":""}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateNotFound(t *testing.T) {
	h := New(&fakeService{err: domain.ErrNotFound}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{"title":"x"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Update(rec, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{"title":"x"}`)).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDeleteSuccess(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Delete(rec, httptest.NewRequest(http.MethodDelete, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteInvalidID(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Delete(rec, httptest.NewRequest(http.MethodDelete, "/tasks/abc", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteNotFound(t *testing.T) {
	h := New(&fakeService{err: domain.ErrNotFound}, nil)
	rec := httptest.NewRecorder()
	h.Delete(rec, httptest.NewRequest(http.MethodDelete, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteInternalError(t *testing.T) {
	h := New(&fakeService{err: errors.New("boom")}, nil)
	rec := httptest.NewRecorder()
	h.Delete(rec, httptest.NewRequest(http.MethodDelete, "/tasks/1", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestOperationsUnauthorized(t *testing.T) {
	h := New(&fakeService{}, nil)
	cases := []struct {
		name string
		call func(http.ResponseWriter, *http.Request)
		req  *http.Request
	}{
		{"create", h.Create, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString("{}"))},
		{"get", h.Get, httptest.NewRequest(http.MethodGet, "/tasks/1", nil)},
		{"update", h.Update, httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString("{}"))},
		{"delete", h.Delete, httptest.NewRequest(http.MethodDelete, "/tasks/1", nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rec, tc.req)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestListNilBecomesEmpty(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil).WithContext(ctxWithClaims()))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "[]")
}

func TestHealthOK(t *testing.T) {
	h := New(&fakeService{}, map[string]func(context.Context) error{
		"postgres": func(context.Context) error { return nil },
		"rabbitmq": func(context.Context) error { return nil },
	})
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthDegraded(t *testing.T) {
	var logs bytes.Buffer
	h := New(&fakeService{}, map[string]func(context.Context) error{
		"rabbitmq": func(context.Context) error { return errors.New("down") },
	}, slog.New(slog.NewJSONHandler(&logs, nil)))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(correlation.NewContext(context.Background(), "cid-health"))
	h.Health(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, logs.String(), `"correlation_id":"cid-health"`)
}

func TestRoutesAuth(t *testing.T) {
	h := New(&fakeService{tasks: []domain.Task{}}, nil)
	rr := h.Routes("secret")

	rec := httptest.NewRecorder()
	rr.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token(t, "secret"))
	rec = httptest.NewRecorder()
	rr.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRoutesHealth(t *testing.T) {
	h := New(&fakeService{}, nil)
	rec := httptest.NewRecorder()
	h.Routes("secret").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}
