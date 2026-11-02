package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/platform/app"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/telemetry"
)

func main() {
	os.Exit(mainWithContext(context.Background()))
}

func mainWithContext(ctx context.Context) int {
	log := logger.New()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return run(ctx, log, defaultRuntimeDeps())
}

type runtimeDeps struct {
	load           func() (config.Config, error)
	setupTelemetry func(context.Context, string) (telemetry.Shutdown, bool, error)
	runApp         func(context.Context, config.Config, *slog.Logger) error
}

func defaultRuntimeDeps() runtimeDeps {
	return runtimeDeps{
		load: config.Load,
		setupTelemetry: func(ctx context.Context, endpoint string) (telemetry.Shutdown, bool, error) {
			return telemetry.Setup(ctx, endpoint, "service-identity")
		},
		runApp: func(ctx context.Context, cfg config.Config, log *slog.Logger) error {
			deps := app.Deps{OpenDB: func(dsn string) (*sql.DB, error) { return sql.Open("postgres", dsn) }}
			return app.Run(ctx, cfg, log, deps)
		},
	}
}

func run(ctx context.Context, log *slog.Logger, deps runtimeDeps) int {
	cfg, err := deps.load()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		return 1
	}
	shutdownTelemetry, enabled, err := deps.setupTelemetry(ctx, cfg.OTLPEndpoint)
	if err != nil {
		log.Error("initialize telemetry", "error", err)
		return 1
	}
	if !enabled {
		log.Info("telemetry exporter disabled", "reason", "OTEL_EXPORTER_OTLP_ENDPOINT is empty")
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error("shutdown telemetry", "error", err)
		}
	}()

	if err := deps.runApp(ctx, cfg, log); err != nil {
		log.Error("service stopped", "error", err)
		return 1
	}
	return 0
}
