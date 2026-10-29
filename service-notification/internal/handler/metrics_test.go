package handler

import (
	"errors"
	"log/slog"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type fakeCounter struct{ labels [][]string }

func (f *fakeCounter) Inc(labels ...string) { f.labels = append(f.labels, labels) }

func TestConsumerMetricsRecordAckedAndRejectedEvents(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "acked", body: `{"task_id":1,"title":"task"}`, want: "acked"},
		{name: "rejected", body: `{`, want: "rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			consumer := NewConsumer(&fakeChannel{}, slog.Default(), counter)
			delivery := amqp.Delivery{Acknowledger: &fakeAck{}, Body: []byte(tt.body)}

			require.NoError(t, consumer.handle(delivery))
			require.Equal(t, [][]string{{tt.want}}, counter.labels)
		})
	}
}

func TestConsumerMetricsRecordConfirmationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "ack", body: `{"task_id":1,"title":"task"}`, want: "ack_error"},
		{name: "nack", body: `{`, want: "nack_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			consumer := NewConsumer(&fakeChannel{}, slog.Default(), counter)
			delivery := amqp.Delivery{Acknowledger: &fakeAck{err: errors.New("confirm failed")}, Body: []byte(tt.body)}

			require.Error(t, consumer.handle(delivery))
			require.Equal(t, [][]string{{tt.want}}, counter.labels)
		})
	}
}
