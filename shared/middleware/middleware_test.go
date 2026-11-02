package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCorrelation(t *testing.T) {
	const (
		validHeader = "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d"
		validCtx    = "0b3c3b8e-1f9a-4f3e-9d39-2f6a0c8f5d21"
	)
	tests := []struct {
		name        string
		header      string
		ctxID       string
		want        string
		wantNewUUID bool
	}{
		{name: "keeps valid header", header: validHeader, want: validHeader},
		{name: "generates when header missing", wantNewUUID: true},
		{name: "replaces invalid header", header: "not-a-uuid", wantNewUUID: true},
		{name: "replaces non-v4 header", header: "6ba7b810-9dad-11d1-80b4-00c04fd430c8", wantNewUUID: true},
		{name: "keeps valid id from context", ctxID: validCtx, want: validCtx},
		{name: "replaces invalid id from context", ctxID: "outer-cid", wantNewUUID: true},
		{name: "prefers valid header over context", header: validHeader, ctxID: validCtx, want: validHeader},
		{name: "falls back to context when header invalid", header: "not-a-uuid", ctxID: validCtx, want: validCtx},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got string
			h := Correlation(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = correlation.FromContext(r.Context())
			}))
			ctx := context.Background()
			if tt.ctxID != "" {
				ctx = correlation.NewContext(ctx, tt.ctxID)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			if tt.header != "" {
				req.Header.Set(HeaderCorrelationID, tt.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if tt.wantNewUUID {
				parsed, err := uuid.Parse(got)
				require.NoError(t, err)
				require.Equal(t, uuid.Version(4), parsed.Version())
				require.NotEqual(t, tt.header, got)
				require.NotEqual(t, tt.ctxID, got)
			} else {
				require.Equal(t, tt.want, got)
			}
			require.Equal(t, got, rec.Header().Get(HeaderCorrelationID))
		})
	}
}

func TestCorrelationAddsIDToActiveSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	ctx, span := provider.Tracer("test").Start(context.Background(), "request")
	h := Correlation(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	req.Header.Set(HeaderCorrelationID, "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d")

	h.ServeHTTP(httptest.NewRecorder(), req)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Contains(t, spans[0].Attributes, attribute.String("correlation_id", req.Header.Get(HeaderCorrelationID)))
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

func TestLoggingSanitizesJSONBodyWithoutConsumingIt(t *testing.T) {
	var logs bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&logs, nil))
	body := `{"email":"person@example.com","password":"secret","name":"Person","title":"Private title","description":"Private description","status":"doing","priority":"urgent","due_date":"2026-11-01","before_id":12,"unknown":"private"}`
	var handledBody string
	h := Logging(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		handledBody = string(payload)
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	h.ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, body, handledBody)
	entry := decodeLogEntry(t, &logs)
	sanitized := entry["request_body"].(map[string]any)
	require.Equal(t, "[REDACTED]", sanitized["email"])
	require.Equal(t, "[REDACTED]", sanitized["password"])
	require.Equal(t, "[REDACTED]", sanitized["name"])
	require.Equal(t, "[REDACTED]", sanitized["title"])
	require.Equal(t, "[REDACTED]", sanitized["description"])
	require.Equal(t, "[REDACTED]", sanitized["unknown"])
	require.Equal(t, "doing", sanitized["status"])
	require.Equal(t, "urgent", sanitized["priority"])
	require.Equal(t, "2026-11-01", sanitized["due_date"])
	require.Equal(t, float64(12), sanitized["before_id"])
	require.Equal(t, float64(len(body)), entry["request_body_bytes"])
	require.Equal(t, false, entry["request_body_truncated"])
	require.Equal(t, true, entry["request_body_valid_json"])
	require.NotContains(t, logs.String(), "person@example.com")
	require.NotContains(t, logs.String(), "Private title")
}

func TestLoggingDoesNotExposeInvalidJSON(t *testing.T) {
	var logs bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&logs, nil))
	body := `{"password":"must-not-leak"`
	h := Logging(l)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeLogEntry(t, &logs)
	require.Equal(t, false, entry["request_body_valid_json"])
	require.NotContains(t, entry, "request_body")
	require.NotContains(t, logs.String(), "must-not-leak")
}

func TestLoggingDoesNotExposeTruncatedJSON(t *testing.T) {
	var logs bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&logs, nil))
	body := `{"description":"` + strings.Repeat("private", 10_000) + `"}`
	h := Logging(l)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodPut, "/tasks/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeLogEntry(t, &logs)
	require.Equal(t, true, entry["request_body_truncated"])
	require.NotContains(t, entry, "request_body")
	require.NotContains(t, logs.String(), strings.Repeat("private", 20))
}

func TestLoggingRecordsOnlyMetadataForNonJSONBody(t *testing.T) {
	var logs bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&logs, nil))
	h := Logging(l)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
	}))
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("must-not-leak"))
	req.Header.Set("Content-Type", "text/plain")
	h.ServeHTTP(httptest.NewRecorder(), req)

	entry := decodeLogEntry(t, &logs)
	require.Equal(t, float64(len("must-not-leak")), entry["request_body_bytes"])
	require.NotContains(t, entry, "request_body")
	require.NotContains(t, logs.String(), "must-not-leak")
}

func decodeLogEntry(t *testing.T, logs *bytes.Buffer) map[string]any {
	t.Helper()
	entry := map[string]any{}
	require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
	return entry
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

func TestRateLimiterReportsRejectedRequest(t *testing.T) {
	limited := 0
	limiter := NewRateLimiter(0, time.Minute, time.Now).OnLimit(func(*http.Request) { limited++ })
	rec := httptest.NewRecorder()

	limiter.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/login", nil),
	)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, 1, limited)
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
