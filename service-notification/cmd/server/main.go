package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/app"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
)

func main() {
	os.Exit(mainWithContext(context.Background()))
}

func mainWithContext(ctx context.Context) int {
	log := logger.New("service-notification")
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, config.Load(), log, dialRabbit); err != nil {
		log.Error("service stopped", "error", err)
		return 1
	}
	return 0
}

type connection interface {
	Channel() (*amqp.Channel, error)
	Close() error
	IsClosed() bool
}

type amqpConn struct {
	connection
}

// Channel abre um canal AMQP compatível com app.Channel.
func (c amqpConn) Channel() (app.Channel, error) {
	return c.connection.Channel()
}

var dialAMQP = amqp.Dial

func dialRabbit(url string) (app.Conn, error) {
	conn, err := dialAMQP(url)
	if err != nil {
		return nil, err
	}
	return amqpConn{connection: conn}, nil
}
