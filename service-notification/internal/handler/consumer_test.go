package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/broker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	apitrace "go.opentelemetry.io/otel/trace"
)

type fakeChannel struct {
	deliveries   chan amqp.Delivery
	err          error
	declareErr   error
	failAt       string
	exchanges    []string
	queues       []string
	queueArgs    []amqp.Table
	binds        int
	consumeCalls int
	exchangeN    int
	queueN       int
	bindN        int
	prefetch     int
	qosErr       error
	started      chan struct{}
}

func (f *fakeChannel) fail(stage string) error {
	if f.declareErr != nil {
		return f.declareErr
	}
	if f.failAt == stage {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	f.exchanges = append(f.exchanges, name)
	f.exchangeN++
	return f.fail("exchange" + string(rune('0'+f.exchangeN)))
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	f.queues = append(f.queues, name)
	f.queueArgs = append(f.queueArgs, args)
	f.queueN++
	return amqp.Queue{Name: name}, f.fail("queue" + string(rune('0'+f.queueN)))
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	f.binds++
	f.bindN++
	return f.fail("bind" + string(rune('0'+f.bindN)))
}

func (f *fakeChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	f.consumeCalls++
	if f.started != nil {
		close(f.started)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.deliveries, nil
}

func (f *fakeChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	f.prefetch = prefetchCount
	return f.qosErr
}

type fakeAck struct {
	acked   int
	nacked  int
	reject  int
	requeue bool
	err     error
}

func (f *fakeAck) Ack(tag uint64, multiple bool) error {
	f.acked++
	return f.err
}

func (f *fakeAck) Nack(tag uint64, multiple, requeue bool) error {
	f.nacked++
	f.requeue = requeue
	return f.err
}

func (f *fakeAck) Reject(tag uint64, requeue bool) error {
	f.reject++
	return nil
}

func newTestConsumer(ch Channel, log *slog.Logger) *Consumer {
	return NewConsumer(ch, log, &fakeCounter{})
}

func TestStartProcessesValidEvent(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
	ack := &fakeAck{}
	ch.deliveries <- amqp.Delivery{
		Acknowledger: ack,
		Headers:      amqp.Table{"x-correlation-id": "646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d"},
		Body:         []byte(`{"task_id":1,"title":"t"}`),
	}
	close(ch.deliveries)

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	c := newTestConsumer(ch, log)
	err := c.Start(context.Background())
	require.ErrorContains(t, err, "delivery channel closed")
	require.Equal(t, 1, ack.acked)
	require.Contains(t, buf.String(), `"correlation_id":"646a4ca0-94d8-4f4a-b0f0-31fb0ac7b31d"`)
	require.Contains(t, buf.String(), `"task_id":1`)
}

func TestStartInvalidEvent(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
	ack := &fakeAck{}
	ch.deliveries <- amqp.Delivery{Acknowledger: ack, Body: []byte(`{`)}
	close(ch.deliveries)

	var buf bytes.Buffer
	c := newTestConsumer(ch, slog.New(slog.NewJSONHandler(&buf, nil)))
	require.ErrorContains(t, c.Start(context.Background()), "delivery channel closed")
	require.Equal(t, 1, ack.nacked)
	require.False(t, ack.requeue)
	require.Contains(t, buf.String(), "invalid event")
}

func TestStartConsumeError(t *testing.T) {
	c := newTestConsumer(&fakeChannel{err: errors.New("boom")}, slog.Default())
	require.Error(t, c.Start(context.Background()))
}

func TestStartDeclaresTopologyBeforeConsume(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery)}
	close(ch.deliveries)
	require.ErrorContains(t, newTestConsumer(ch, slog.Default()).Start(context.Background()), "delivery channel closed")

	require.Equal(t, []string{"tasks.dlx", "tasks"}, ch.exchanges)
	require.Equal(t, []string{"task.events.dlq", "task.events"}, ch.queues)
	require.Equal(t, 2, ch.binds)
	require.Equal(t, 1, ch.consumeCalls)
	require.Equal(t, 10, ch.prefetch)
	require.Equal(t, amqp.Table{"x-dead-letter-exchange": "tasks.dlx"}, ch.queueArgs[1])
}

func TestStartDeclareError(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery), declareErr: errors.New("boom")}
	err := newTestConsumer(ch, slog.Default()).Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "declare")
	require.Zero(t, ch.consumeCalls)
}

func TestStartSetupStagesFail(t *testing.T) {
	for _, stage := range []string{"exchange1", "queue1", "bind1", "exchange2", "queue2", "bind2"} {
		t.Run(stage, func(t *testing.T) {
			ch := &fakeChannel{deliveries: make(chan amqp.Delivery), failAt: stage}
			err := newTestConsumer(ch, slog.Default()).Start(context.Background())
			require.Error(t, err)
			require.Zero(t, ch.consumeCalls)
		})
	}
}

func TestStartContextCanceled(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, newTestConsumer(ch, slog.Default()).Start(ctx))
}

func TestStartClosedChannel(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery)}
	close(ch.deliveries)
	require.ErrorContains(t, newTestConsumer(ch, slog.Default()).Start(context.Background()), "delivery channel closed")
}

func TestStartInvalidCorrelationIDGeneratesUUID(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
	ack := &fakeAck{}
	ch.deliveries <- amqp.Delivery{
		Acknowledger: ack,
		Headers:      amqp.Table{broker.CorrelationIDHeader: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		Body:         []byte(`{"task_id":2,"title":"task"}`),
	}
	close(ch.deliveries)
	var buf bytes.Buffer
	require.Error(t, newTestConsumer(ch, slog.New(slog.NewJSONHandler(&buf, nil))).Start(context.Background()))
	require.Equal(t, 1, ack.acked)

	var entry map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if bytes.Contains(line, []byte(`"msg":"notification_received"`)) {
			require.NoError(t, json.Unmarshal(line, &entry))
		}
	}
	parsed, err := uuid.Parse(entry["correlation_id"].(string))
	require.NoError(t, err)
	require.Equal(t, uuid.Version(4), parsed.Version())
}

func TestStartRejectsSemanticallyInvalidEvents(t *testing.T) {
	for _, body := range []string{
		`{"task_id":0,"title":"task"}`,
		`{"task_id":1,"title":"   "}`,
	} {
		t.Run(body, func(t *testing.T) {
			ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
			ack := &fakeAck{}
			ch.deliveries <- amqp.Delivery{Acknowledger: ack, Body: []byte(body)}
			close(ch.deliveries)

			require.Error(t, newTestConsumer(ch, slog.Default()).Start(context.Background()))
			require.Equal(t, 1, ack.nacked)
			require.Zero(t, ack.acked)
		})
	}
}

func TestStartReturnsAckAndNackErrors(t *testing.T) {
	for name, body := range map[string]string{
		"ack":           `{"task_id":1,"title":"task"}`,
		"nack decode":   `{`,
		"nack semantic": `{"task_id":0,"title":"task"}`,
	} {
		t.Run(name, func(t *testing.T) {
			ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
			ch.deliveries <- amqp.Delivery{Acknowledger: &fakeAck{err: errors.New("confirm failed")}, Body: []byte(body)}

			err := newTestConsumer(ch, slog.Default()).Start(context.Background())
			if name == "ack" {
				require.ErrorContains(t, err, "ack delivery")
			} else {
				require.ErrorContains(t, err, "nack delivery")
			}
		})
	}
}

func TestStartReturnsQosError(t *testing.T) {
	ch := &fakeChannel{qosErr: errors.New("qos failed")}
	require.ErrorContains(t, newTestConsumer(ch, slog.Default()).Start(context.Background()), "set qos")
	require.Zero(t, ch.consumeCalls)
}

func TestConsumerActiveOnlyWhileConsuming(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	ch := &fakeChannel{deliveries: deliveries}
	consumer := newTestConsumer(ch, slog.Default())
	done := make(chan error, 1)
	go func() { done <- consumer.Start(context.Background()) }()

	require.Eventually(t, consumer.Active, time.Second, time.Millisecond)
	close(deliveries)
	require.Error(t, <-done)
	require.False(t, consumer.Active())
}

func TestConsumerExtractsW3CTraceContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	oldProvider := otel.GetTracerProvider()
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
		otel.SetTextMapPropagator(oldPropagator)
		require.NoError(t, tp.Shutdown(context.Background()))
	})

	ch := &fakeChannel{deliveries: make(chan amqp.Delivery, 1)}
	ch.deliveries <- amqp.Delivery{
		Acknowledger: &fakeAck{},
		Headers: amqp.Table{
			"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		},
		Body: []byte(`{"task_id":1,"title":"task"}`),
	}
	close(ch.deliveries)
	require.Error(t, newTestConsumer(ch, slog.Default()).Start(context.Background()))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "task.created consume", spans[0].Name)
	require.Equal(t, apitrace.SpanKindConsumer, spans[0].SpanKind)
	require.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spans[0].Parent.TraceID().String())
	require.Equal(t, "00f067aa0ba902b7", spans[0].Parent.SpanID().String())
	require.Equal(t, spans[0].Parent.TraceID(), spans[0].SpanContext.TraceID())
}

func TestAMQPHeaderCarrier(t *testing.T) {
	carrier := broker.HeaderCarrier{}
	carrier.Set("traceparent", "value")
	require.Equal(t, "value", carrier.Get("TraceParent"))
	require.Empty(t, carrier.Get("tracestate"))
	require.Equal(t, []string{"traceparent"}, carrier.Keys())
}

func TestConsumerLogsLifecycle(t *testing.T) {
	ch := &fakeChannel{deliveries: make(chan amqp.Delivery), started: make(chan struct{})}
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- newTestConsumer(ch, log).Start(ctx) }()
	<-ch.started
	cancel()
	require.NoError(t, <-done)
	require.Contains(t, buf.String(), `"msg":"consumer_started"`)
	require.Contains(t, buf.String(), `"msg":"consumer_stopped"`)
}
