package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel/trace"
)

func TestNewEmitsServiceName(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(&buf, "service-test")
	l.Info("hello", "k", "v")

	entry := decodeEntry(t, &buf)
	require.Equal(t, "service-test", entry["service_name"])
	require.Equal(t, "hello", entry["msg"])
	require.Equal(t, "v", entry["k"])
}

func TestWithContextAddsCorrelationAndTrace(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{2},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	ctx = correlation.NewContext(ctx, "cid-1")

	WithContext(ctx, base).InfoContext(ctx, "x")

	entry := decodeEntry(t, &buf)
	require.Equal(t, "cid-1", entry["correlation_id"])
	require.Equal(t, spanContext.TraceID().String(), entry["trace_id"])
	require.Equal(t, spanContext.SpanID().String(), entry["span_id"])
}

func TestWithContextOmitsUnavailableIdentifiers(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	WithContext(context.Background(), base).Info("x")

	entry := decodeEntry(t, &buf)
	require.NotContains(t, entry, "correlation_id")
	require.NotContains(t, entry, "trace_id")
	require.NotContains(t, entry, "span_id")
}

func TestNewNotNil(t *testing.T) {
	require.NotNil(t, New("service-test"))
}

func decodeEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	entry := map[string]any{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	return entry
}
