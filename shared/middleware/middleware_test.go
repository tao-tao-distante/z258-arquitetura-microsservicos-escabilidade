package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

func TestCorrelationExistingHeader(t *testing.T) {
	var got string
	h := Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderCorrelationID, "abc")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, "abc", got)
	require.Equal(t, "abc", rec.Header().Get(HeaderCorrelationID))
}

func TestCorrelationGeneratesWhenMissing(t *testing.T) {
	var got string
	h := Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.NotEmpty(t, got)
	require.Equal(t, got, rec.Header().Get(HeaderCorrelationID))
}

func TestCorrelationPreservesIDAlreadyInContext(t *testing.T) {
	var got string
	h := Correlation(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
		correlation.NewContext(context.Background(), "outer-cid"),
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, "outer-cid", got)
	require.Equal(t, "outer-cid", rec.Header().Get(HeaderCorrelationID))
}

func TestCorrelationUUIDAcceptsValidHeader(t *testing.T) {
	const valid = "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d"
	var got string
	h := CorrelationUUID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderCorrelationID, valid)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, valid, got)
	require.Equal(t, valid, rec.Header().Get(HeaderCorrelationID))
}

func TestCorrelationUUIDGeneratesWhenMissing(t *testing.T) {
	var got string
	h := CorrelationUUID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	_, err := uuid.Parse(got)
	require.NoError(t, err)
	require.Equal(t, got, rec.Header().Get(HeaderCorrelationID))
}

func TestCorrelationUUIDReplacesInvalid(t *testing.T) {
	var got string
	h := CorrelationUUID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = correlation.FromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderCorrelationID, "invalid")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.NotEqual(t, "invalid", got)
	_, err := uuid.Parse(got)
	require.NoError(t, err)
	require.Equal(t, got, rec.Header().Get(HeaderCorrelationID))
}

func TestLoggingEmitsFields(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Logging(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil).WithContext(
		correlation.NewContext(context.Background(), "cid"),
	)
	h.ServeHTTP(httptest.NewRecorder(), req)
	out := buf.String()
	require.Contains(t, out, `"path":"/tasks"`)
	require.Contains(t, out, `"method":"POST"`)
	require.Contains(t, out, `"status_code":201`)
	require.Contains(t, out, `"latency_ms"`)
	require.Contains(t, out, `"correlation_id":"cid"`)
}

func TestStatusRecorderDefault(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: 200}
	rec.WriteHeader(http.StatusAccepted)
	require.Equal(t, http.StatusAccepted, rec.status)
}

func TestRateLimiterRejectsExcessRequestsAndResets(t *testing.T) {
	now := time.Unix(1, 0)
	limiter := NewRateLimiter(2, time.Minute, func() time.Time { return now })
	h := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for i, want := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		h.ServeHTTP(rec, req)
		require.Equal(t, want, rec.Code, "request %d", i+1)
	}
	now = now.Add(time.Minute)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRateLimiterSeparatesClients(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute, time.Now)
	h := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	for _, addr := range []string{"192.0.2.1:1", "192.0.2.2:1"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/register", nil)
		req.RemoteAddr = addr
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimiterHandlesRemoteAddressWithoutPort(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute, time.Now)
	h := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "local-client"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
