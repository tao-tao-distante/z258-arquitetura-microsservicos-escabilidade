package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

func TestNewEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	l.Info("hello", "k", "v")
	require.Contains(t, buf.String(), `"msg":"hello"`)
	require.Contains(t, buf.String(), `"k":"v"`)
}

func TestWithCorrelationPresent(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := correlation.NewContext(context.Background(), "cid-1")
	WithCorrelation(ctx, base).Info("x")
	require.Contains(t, buf.String(), `"correlation_id":"cid-1"`)
}

func TestWithCorrelationAbsent(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))
	WithCorrelation(context.Background(), base).Info("x")
	require.NotContains(t, buf.String(), "correlation_id")
}

func TestNewNotNil(t *testing.T) {
	require.NotNil(t, New())
}
