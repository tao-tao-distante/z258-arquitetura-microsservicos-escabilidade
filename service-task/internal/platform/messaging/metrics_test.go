package messaging

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/broker"
)

type fakeCounter struct{ labels [][]string }

func (f *fakeCounter) Inc(labels ...string) { f.labels = append(f.labels, labels) }

func TestDispatcherMetricsRecordSentAndRetry(t *testing.T) {
	tests := []struct {
		name      string
		publisher fakeReliablePublisher
		want      string
	}{
		{name: "sent", want: "sent"},
		{name: "retry", publisher: fakeReliablePublisher{err: errors.New("rabbit down")}, want: "retry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			store := &fakeOutbox{event: &repository.OutboxEvent{ID: 1, EventID: "event", EventType: broker.RoutingKey}}
			dispatcher := NewDispatcher(store, tt.publisher, slog.Default(), counter)

			require.NoError(t, dispatcher.DispatchOne(context.Background()))
			require.Equal(t, [][]string{{tt.want}}, counter.labels)
		})
	}
}

func TestDispatcherMetricsRecordStoreErrors(t *testing.T) {
	tests := []struct {
		name      string
		store     *fakeOutbox
		publisher fakeReliablePublisher
		want      string
	}{
		{name: "claim", store: &fakeOutbox{claimErr: errors.New("claim")}, want: "claim_error"},
		{name: "mark", store: &fakeOutbox{event: &repository.OutboxEvent{ID: 1}, markErr: errors.New("mark")}, want: "mark_error"},
		{name: "retry", store: &fakeOutbox{event: &repository.OutboxEvent{ID: 1}, retryErr: errors.New("retry")}, publisher: fakeReliablePublisher{err: errors.New("publish")}, want: "retry_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter := &fakeCounter{}
			dispatcher := NewDispatcher(tt.store, tt.publisher, slog.Default(), counter)

			require.Error(t, dispatcher.DispatchOne(context.Background()))
			require.Equal(t, [][]string{{tt.want}}, counter.labels)
		})
	}
}
