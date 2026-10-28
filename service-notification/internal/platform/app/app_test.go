package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/config"
)

type fakeChannel struct {
	deliveries chan amqp.Delivery
	err        error
	closed     bool
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return nil
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, nil
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (f *fakeChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.deliveries, nil
}

func (f *fakeChannel) Qos(int, int, bool) error { return nil }

func (f *fakeChannel) Close() error   { return nil }
func (f *fakeChannel) IsClosed() bool { return f.closed }

type fakeConn struct {
	ch     Channel
	err    error
	closed bool
}

func (f *fakeConn) Channel() (Channel, error) { return f.ch, f.err }
func (f *fakeConn) Close() error              { return nil }
func (f *fakeConn) IsClosed() bool            { return f.closed }

func TestRunSuccess(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	conn := &fakeConn{ch: &fakeChannel{deliveries: deliveries}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(ctx, config.Config{Port: "0", RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return conn, nil
	})
	require.NoError(t, err)
}

func TestRunDialError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(ctx, config.Config{RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return nil, errors.New("boom")
	})
	require.Error(t, err)
}

func TestRunChannelError(t *testing.T) {
	conn := &fakeConn{err: errors.New("boom")}
	err := Run(context.Background(), config.Config{RabbitURL: "x"}, slog.Default(), func(string) (Conn, error) {
		return conn, nil
	})
	require.Error(t, err)
}

func TestNewServer(t *testing.T) {
	srv := NewServer("9999", http.NewServeMux())
	require.Equal(t, ":9999", srv.Addr)
	require.NotNil(t, srv.Handler)
	require.NotZero(t, srv.ReadHeaderTimeout)
	require.NotZero(t, srv.ReadTimeout)
	require.NotZero(t, srv.WriteTimeout)
	require.NotZero(t, srv.IdleTimeout)
}

func TestShutdown(t *testing.T) {
	require.NoError(t, Shutdown(NewServer("0", http.NewServeMux())))
}
