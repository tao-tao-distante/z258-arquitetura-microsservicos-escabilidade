package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainWithContextCanceled(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, 1, mainWithContext(ctx))
}

func TestMainWithContextRejectsMissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	require.Equal(t, 1, mainWithContext(context.Background()))
}

func TestDialRabbitRejectsInvalidURL(t *testing.T) {
	_, err := dialRabbit("://invalid")
	require.Error(t, err)
}
