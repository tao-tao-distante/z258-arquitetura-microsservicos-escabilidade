package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

type fakeCounter struct{ labels [][]string }

func (f *fakeCounter) Inc(labels ...string) { f.labels = append(f.labels, labels) }

func TestTaskMetricsRecordCreateOutcome(t *testing.T) {
	counter := &fakeCounter{}
	h := newTestHandlerWithCounter(&fakeService{task: &domain.Task{ID: 1, Title: "task"}}, nil, counter)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"task"}`)).WithContext(ctxWithClaims())

	h.Create(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, [][]string{{"create", "success"}}, counter.labels)
}

func TestTaskMetricsRecordEveryBusinessOperation(t *testing.T) {
	tests := []struct {
		name      string
		service   *fakeService
		request   *http.Request
		invoke    func(*Handler, http.ResponseWriter, *http.Request)
		wantLabel []string
	}{
		{name: "list success", service: &fakeService{}, request: httptest.NewRequest(http.MethodGet, "/tasks", nil).WithContext(ctxWithClaims()), invoke: (*Handler).List, wantLabel: []string{"list", "success"}},
		{name: "get not found", service: &fakeService{err: domain.ErrNotFound}, request: httptest.NewRequest(http.MethodGet, "/tasks/1", nil).WithContext(ctxWithClaims()), invoke: (*Handler).Get, wantLabel: []string{"get", "not_found"}},
		{name: "update invalid", service: &fakeService{err: domain.ErrInvalidInput}, request: httptest.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBufferString(`{"title":""}`)).WithContext(ctxWithClaims()), invoke: (*Handler).Update, wantLabel: []string{"update", "invalid_input"}},
		{name: "delete error", service: &fakeService{err: errors.New("database down")}, request: httptest.NewRequest(http.MethodDelete, "/tasks/1", nil).WithContext(ctxWithClaims()), invoke: (*Handler).Delete, wantLabel: []string{"delete", "error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			h := newTestHandlerWithCounter(tt.service, nil, counter)
			rec := httptest.NewRecorder()

			tt.invoke(h, rec, tt.request)

			require.Equal(t, [][]string{tt.wantLabel}, counter.labels)
		})
	}
}

func TestTaskMetricsRecordJWTRejection(t *testing.T) {
	counter := &fakeCounter{}
	h := newTestHandlerWithCounter(&fakeService{}, nil, counter)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, [][]string{{"list", "unauthorized"}}, counter.labels)
}
