package messaging

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/broker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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

type traceCapturingPublisher struct{ spanContext trace.SpanContext }

func (p *traceCapturingPublisher) Publish(ctx context.Context, _ string, _ string, _ []byte) error {
	p.spanContext = trace.SpanContextFromContext(ctx)
	return nil
}

func TestDispatcherMarksConfirmedEventSent(t *testing.T) {
	store := &fakeOutbox{event: &repository.OutboxEvent{ID: 7, EventID: "event", EventType: broker.RoutingKey, Payload: []byte(`{}`)}}
	d := NewDispatcher(store, fakeReliablePublisher{}, slog.Default())
	require.NoError(t, d.DispatchOne(context.Background()))
	require.Equal(t, int64(7), store.sent)
	require.Zero(t, store.retried)
}

func TestDispatcherRetriesRabbitFailure(t *testing.T) {
	store := &fakeOutbox{event: &repository.OutboxEvent{ID: 7, EventID: "event", EventType: broker.RoutingKey, Payload: []byte(`{}`), Attempts: 2}}
	d := NewDispatcher(store, fakeReliablePublisher{err: errors.New("rabbit down")}, slog.Default())
	require.NoError(t, d.DispatchOne(context.Background()))
	require.Equal(t, int64(7), store.retried)
	require.Zero(t, store.sent)
}

func TestDispatcherRestoresPersistedTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	oldProvider := otel.GetTracerProvider()
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagator)
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	store := &fakeOutbox{event: &repository.OutboxEvent{
		ID:        7,
		EventID:   "event",
		EventType: broker.RoutingKey,
		TraceContext: map[string]string{
			"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		},
	}}
	publisher := &traceCapturingPublisher{}

	require.NoError(t, NewDispatcher(store, publisher, slog.Default()).DispatchOne(context.Background()))
	require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", publisher.spanContext.TraceID().String())
	require.NotEqual(t, "00f067aa0ba902b7", publisher.spanContext.SpanID().String())
	require.False(t, publisher.spanContext.IsRemote())

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "outbox dispatch", spans[0].Name)
	require.Equal(t, trace.SpanKindInternal, spans[0].SpanKind)
	require.Equal(t, "00f067aa0ba902b7", spans[0].Parent.SpanID().String())
	require.Equal(t, spans[0].SpanContext.SpanID(), publisher.spanContext.SpanID())
}

func TestDispatcherPropagatesStoreFailure(t *testing.T) {
	d := NewDispatcher(&fakeOutbox{claimErr: errors.New("db down")}, fakeReliablePublisher{}, slog.Default())
	require.Error(t, d.DispatchOne(context.Background()))
}

func TestDispatcherPropagatesMarkAndRetryFailures(t *testing.T) {
	event := &repository.OutboxEvent{ID: 7, EventID: "event", EventType: broker.RoutingKey}
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
