package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/telemetry"
)

func TestMainWithContextCanceled(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, 1, mainWithContext(ctx))
}

func TestRunRejectsTelemetrySetupError(t *testing.T) {
	deps := runtimeDeps{
		load: func() (config.Config, error) { return config.Config{JWTSecret: "secret"}, nil },
		setupTelemetry: func(context.Context, string) (telemetry.Shutdown, bool, error) {
			return nil, false, errors.New("otel failed")
		},
	}
	require.Equal(t, 1, run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), deps))
}

func TestRunRejectsInvalidConfiguration(t *testing.T) {
	deps := runtimeDeps{
		load: func() (config.Config, error) { return config.Config{}, errors.New("invalid config") },
	}
	require.Equal(t, 1, run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), deps))
}

func TestRunAppFailureAndTelemetryShutdown(t *testing.T) {
	shutdownCalled := false
	deps := runtimeDeps{
		load: func() (config.Config, error) { return config.Config{JWTSecret: "secret"}, nil },
		setupTelemetry: func(context.Context, string) (telemetry.Shutdown, bool, error) {
			return func(context.Context) error { shutdownCalled = true; return nil }, false, nil
		},
		runApp: func(context.Context, config.Config, *slog.Logger) error { return errors.New("app failed") },
	}
	require.Equal(t, 1, run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), deps))
	require.True(t, shutdownCalled)
}

func TestRunSuccess(t *testing.T) {
	deps := runtimeDeps{
		load: func() (config.Config, error) { return config.Config{JWTSecret: "secret"}, nil },
		setupTelemetry: func(context.Context, string) (telemetry.Shutdown, bool, error) {
			return func(context.Context) error { return errors.New("shutdown failed") }, true, nil
		},
		runApp: func(context.Context, config.Config, *slog.Logger) error { return nil },
	}
	require.Equal(t, 0, run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), deps))
}
