package main

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestMainWithContextCanceled(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "invalid://")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, 1, mainWithContext(ctx))
}

type fakeConnection struct{}

func (fakeConnection) Channel() (*amqp.Channel, error) { return nil, errors.New("channel failed") }
func (fakeConnection) Close() error                    { return nil }
func (fakeConnection) IsClosed() bool                  { return false }

func TestAMQPConnChannelPropagatesError(t *testing.T) {
	_, err := amqpConn{connection: fakeConnection{}}.Channel()
	require.ErrorContains(t, err, "channel failed")
}
