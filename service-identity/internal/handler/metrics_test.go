package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
)

type fakeCounter struct{ labels [][]string }

func (f *fakeCounter) Inc(labels ...string) { f.labels = append(f.labels, labels) }

func TestAuthMetricsRecordOperationOutcome(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		service   *fakeService
		wantCode  int
		wantLabel []string
	}{
		{name: "register success", path: "/register", service: &fakeService{user: &domain.User{ID: 1}, token: "token"}, wantCode: http.StatusCreated, wantLabel: []string{"register", "success"}},
		{name: "login denied", path: "/login", service: &fakeService{err: domain.ErrInvalidCredentials}, wantCode: http.StatusUnauthorized, wantLabel: []string{"login", "invalid_credentials"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			h := New(tt.service, nil).WithAuthMetrics(counter)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(`{"email":"a@b.com","password":"password"}`))
			h.Routes().ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			require.Equal(t, [][]string{tt.wantLabel}, counter.labels)
		})
	}
}

func TestAuthMetricsRecordRateLimitedOutcome(t *testing.T) {
	counter := &fakeCounter{}
	h := New(&fakeService{user: &domain.User{ID: 1}, token: "token"}, nil).WithAuthMetrics(counter)

	for range 11 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"a@b.com","password":"password"}`))
		req.RemoteAddr = "192.0.2.10:1234"
		h.Routes().ServeHTTP(rec, req)
	}

	require.Equal(t, []string{"login", "rate_limited"}, counter.labels[len(counter.labels)-1])
}
