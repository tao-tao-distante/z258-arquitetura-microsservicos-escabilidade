package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/handler"
)

func TestReadinessRequiresActiveConsumerAndOpenRabbitMQ(t *testing.T) {
	tests := []struct {
		name           string
		startConsumer  bool
		connectionDown bool
		channelDown    bool
		wantStatus     int
	}{
		{name: "ready", startConsumer: true, wantStatus: http.StatusOK},
		{name: "consumer inactive", wantStatus: http.StatusServiceUnavailable},
		{name: "connection closed", startConsumer: true, connectionDown: true, wantStatus: http.StatusServiceUnavailable},
		{name: "channel closed", startConsumer: true, channelDown: true, wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deliveries := make(chan amqp.Delivery)
			ch := &fakeChannel{deliveries: deliveries, closed: tt.channelDown}
			conn := &fakeConn{ch: ch, closed: tt.connectionDown}
			consumer := handler.NewConsumer(ch, slog.Default())
			ctx, cancel := context.WithCancel(context.Background())
			if tt.startConsumer {
				done := make(chan error, 1)
				go func() { done <- consumer.Start(ctx) }()
				require.Eventually(t, consumer.Active, time.Second, time.Millisecond)
				t.Cleanup(func() {
					cancel()
					require.NoError(t, <-done)
				})
			} else {
				defer cancel()
			}

			recorder := httptest.NewRecorder()
			newHealth(conn, ch, consumer).Health(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		})
	}
}
