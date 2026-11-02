package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/config"
	platformmetrics "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/metrics"
)

type fakeCounter struct{}

func (*fakeCounter) Inc(...string) {}

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

func TestNewHTTPHandlerCorrelation(t *testing.T) {
	const valid = "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d"
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "keeps valid header", header: valid, want: valid},
		{name: "replaces invalid header", header: "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHTTPHandler(map[string]func(context.Context) error{}, slog.Default(), platformmetrics.New("test"))
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.Header.Set("x-correlation-id", tt.header)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			got := rec.Header().Get("x-correlation-id")
			if tt.want != "" {
				require.Equal(t, tt.want, got)
				return
			}
			_, err := uuid.Parse(got)
			require.NoError(t, err)
			require.NotEqual(t, tt.header, got)
		})
	}
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
