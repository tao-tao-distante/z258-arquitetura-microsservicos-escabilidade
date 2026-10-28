package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/app"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/config"
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

	deps := app.Deps{
		OpenDB:     func(dsn string) (*sql.DB, error) { return sql.Open("postgres", dsn) },
		DialRabbit: dialRabbit,
	}
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err, "correlation_id", "")
		return 1
	}
	shutdownTelemetry, _, err := telemetry.Setup(ctx, cfg.OTLPEndpoint, "service-task")
	if err != nil {
		log.Error("initialize telemetry", "error", err, "correlation_id", "")
		return 1
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error("shutdown telemetry", "error", err, "correlation_id", "")
		}
	}()
	if err := app.Run(ctx, cfg, log, deps); err != nil {
		log.Error("service stopped", "error", err, "correlation_id", "")
		return 1
	}
	return 0
}

type amqpConn struct {
	*amqp.Connection
}

func (c amqpConn) Channel() (app.Channel, error) {
	return c.Connection.Channel()
}

func dialRabbit(url string) (app.Conn, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	return amqpConn{Connection: conn}, nil
}
