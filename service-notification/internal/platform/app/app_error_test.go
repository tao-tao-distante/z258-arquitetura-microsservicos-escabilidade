package app

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/config"
)

func TestRunPropagatesTelemetrySetupError(t *testing.T) {
	err := Run(context.Background(), config.Config{Port: "0", RabbitURL: "x", OTLPEndpoint: "://invalid"}, slog.Default(), func(string) (Conn, error) {
		return &fakeConn{ch: &fakeChannel{deliveries: make(chan amqp.Delivery)}}, nil
	})
	require.ErrorContains(t, err, "setup telemetry")
}

func TestRunPropagatesConsumerError(t *testing.T) {
	conn := &fakeConn{ch: &fakeChannel{err: errors.New("consume down")}}
	err := Run(context.Background(), config.Config{Port: "0", RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return conn, nil
	})
	require.ErrorContains(t, err, "consume down")
}

func TestRunPropagatesClosedDeliveryChannel(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	close(deliveries)
	conn := &fakeConn{ch: &fakeChannel{deliveries: deliveries}}
	err := Run(context.Background(), config.Config{Port: "0", RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return conn, nil
	})
	require.ErrorContains(t, err, "delivery channel closed")
}

func TestRunPropagatesServerError(t *testing.T) {
	conn := &fakeConn{ch: &fakeChannel{deliveries: make(chan amqp.Delivery)}}
	err := Run(context.Background(), config.Config{Port: "invalid", RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return conn, nil
	})
	require.ErrorContains(t, err, "http server")
}
