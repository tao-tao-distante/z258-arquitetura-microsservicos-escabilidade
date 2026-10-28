package messaging

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
)

type fakeOutbox struct {
	event    *repository.OutboxEvent
	claimErr error
	sent     int64
	retried  int64
	markErr  error
	retryErr error
}

func (f *fakeOutbox) ClaimOutbox(context.Context) (*repository.OutboxEvent, error) {
	return f.event, f.claimErr
}
func (f *fakeOutbox) MarkOutboxSent(_ context.Context, id int64) error {
	f.sent = id
	return f.markErr
}
func (f *fakeOutbox) RetryOutbox(_ context.Context, id int64, _ int) error {
	f.retried = id
	return f.retryErr
}

type fakeReliablePublisher struct{ err error }

func (f fakeReliablePublisher) Publish(context.Context, string, string, []byte) error { return f.err }

func TestDispatcherMarksConfirmedEventSent(t *testing.T) {
	store := &fakeOutbox{event: &repository.OutboxEvent{ID: 7, EventID: "event", EventType: RoutingKey, Payload: []byte(`{}`)}}
	d := NewDispatcher(store, fakeReliablePublisher{}, slog.Default())
	require.NoError(t, d.DispatchOne(context.Background()))
	require.Equal(t, int64(7), store.sent)
	require.Zero(t, store.retried)
}

func TestDispatcherRetriesRabbitFailure(t *testing.T) {
	store := &fakeOutbox{event: &repository.OutboxEvent{ID: 7, EventID: "event", EventType: RoutingKey, Payload: []byte(`{}`), Attempts: 2}}
	d := NewDispatcher(store, fakeReliablePublisher{err: errors.New("rabbit down")}, slog.Default())
	require.NoError(t, d.DispatchOne(context.Background()))
	require.Equal(t, int64(7), store.retried)
	require.Zero(t, store.sent)
}

func TestDispatcherPropagatesStoreFailure(t *testing.T) {
	d := NewDispatcher(&fakeOutbox{claimErr: errors.New("db down")}, fakeReliablePublisher{}, slog.Default())
	require.Error(t, d.DispatchOne(context.Background()))
}

func TestDispatcherPropagatesMarkAndRetryFailures(t *testing.T) {
	event := &repository.OutboxEvent{ID: 7, EventID: "event", EventType: RoutingKey}
	require.Error(t, NewDispatcher(&fakeOutbox{event: event, markErr: errors.New("mark")}, fakeReliablePublisher{}, slog.Default()).DispatchOne(context.Background()))
	require.Error(t, NewDispatcher(&fakeOutbox{event: event, retryErr: errors.New("retry")}, fakeReliablePublisher{err: errors.New("rabbit")}, slog.Default()).DispatchOne(context.Background()))
}

func TestDispatcherRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	NewDispatcher(&fakeOutbox{}, fakeReliablePublisher{}, slog.Default()).Run(ctx)
}

func TestDispatcherRunLogsFailureAndTicks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 550*time.Millisecond)
	defer cancel()
	NewDispatcher(&fakeOutbox{claimErr: errors.New("db down")}, fakeReliablePublisher{}, slog.Default()).Run(ctx)
}
