package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type fakeChannel struct {
	declaredExchange   string
	declaredQueue      string
	publishedKey       string
	publishedMsg       amqp.Publishing
	err                error
	failAt             string
	exchangeCalls      int
	queueCalls         int
	bindCalls          int
	mandatory          bool
	confirmErr         error
	confirmations      chan amqp.Confirmation
	returns            chan amqp.Return
	nack               bool
	returned           bool
	noConfirm          bool
	returnAfterConfirm bool
}

func (f *fakeChannel) fail(stage string) error {
	if f.err != nil {
		return f.err
	}
	if f.failAt == stage {
		return errors.New("fail")
	}
	return nil
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	f.declaredExchange = name
	f.exchangeCalls++
	return f.fail("exchange" + itoa(f.exchangeCalls))
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	f.declaredQueue = name
	f.queueCalls++
	return amqp.Queue{Name: name}, f.fail("queue" + itoa(f.queueCalls))
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	f.bindCalls++
	return f.fail("bind" + itoa(f.bindCalls))
}

func (f *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	f.publishedKey = key
	f.publishedMsg = msg
	f.mandatory = mandatory
	if f.err == nil && f.returned && !f.returnAfterConfirm {
		f.returns <- amqp.Return{MessageId: msg.MessageId}
	}
	if f.err == nil && !f.noConfirm {
		f.confirmations <- amqp.Confirmation{Ack: !f.nack}
	}
	if f.err == nil && f.returned && f.returnAfterConfirm {
		go func() {
			time.Sleep(2 * time.Millisecond)
			f.returns <- amqp.Return{MessageId: msg.MessageId}
		}()
	}
	return f.err
}

func (f *fakeChannel) Confirm(bool) error { return f.confirmErr }
func (f *fakeChannel) NotifyPublish(ch chan amqp.Confirmation) chan amqp.Confirmation {
	f.confirmations = ch
	return ch
}
func (f *fakeChannel) NotifyReturn(ch chan amqp.Return) chan amqp.Return {
	f.returns = ch
	return ch
}

func itoa(i int) string {
	return string(rune('0' + i))
}

func TestPublishTaskCreated(t *testing.T) {
	ch := &fakeChannel{}
	p := New(ch)
	require.NoError(t, p.Setup())
	ctx := correlation.NewContext(context.Background(), "cid-123")
	require.NoError(t, p.PublishTaskCreated(ctx, 9, "Minha tarefa"))
	require.Equal(t, "task.created", ch.publishedKey)
	require.Equal(t, "cid-123", ch.publishedMsg.Headers["x-correlation-id"])
	require.Equal(t, "application/json", ch.publishedMsg.ContentType)
	require.Contains(t, string(ch.publishedMsg.Body), "Minha tarefa")
	require.True(t, ch.mandatory)
}

func TestPublishInjectsTraceContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ch := &fakeChannel{}
	p := New(ch)
	require.NoError(t, p.Setup())
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	require.NoError(t, p.PublishTaskCreated(ctx, 1, "x"))
	require.NotEmpty(t, ch.publishedMsg.Headers["traceparent"])
}

func TestPublishTaskCreatedError(t *testing.T) {
	ch := &fakeChannel{}
	p := New(ch)
	require.NoError(t, p.Setup())
	ch.err = amqp.ErrClosed
	require.Error(t, p.PublishTaskCreated(context.Background(), 1, "x"))
}

func TestPublishRejectsNackAndMandatoryReturn(t *testing.T) {
	for _, ch := range []*fakeChannel{{nack: true}, {returned: true}} {
		p := New(ch)
		require.NoError(t, p.Setup())
		require.Error(t, p.PublishTaskCreated(context.Background(), 1, "x"))
	}
}

func TestSetupError(t *testing.T) {
	p := New(&fakeChannel{err: amqp.ErrClosed})
	require.Error(t, p.Setup())
}

func TestSetup(t *testing.T) {
	ch := &fakeChannel{}
	p := New(ch)
	require.NoError(t, p.Setup())
	require.Equal(t, "tasks", ch.declaredExchange)
	require.Equal(t, "task.events", ch.declaredQueue)
}

func TestSetupStagesFail(t *testing.T) {
	for _, stage := range []string{"exchange1", "queue1", "bind1", "exchange2", "queue2", "bind2"} {
		t.Run(stage, func(t *testing.T) {
			p := New(&fakeChannel{failAt: stage})
			require.Error(t, p.Setup())
		})
	}
}

func TestSetupConfirmError(t *testing.T) {
	require.Error(t, New(&fakeChannel{confirmErr: errors.New("confirm")}).Setup())
}

func TestPublishRequiresSetup(t *testing.T) {
	require.Error(t, New(&fakeChannel{}).Publish(context.Background(), "event", RoutingKey, nil))
}

func TestTableCarrier(t *testing.T) {
	carrier := tableCarrier{"traceparent": "value"}
	require.Equal(t, "value", carrier.Get("traceparent"))
	require.Contains(t, carrier.Keys(), "traceparent")
}

func TestPublishStopsOnContextCanceled(t *testing.T) {
	ch := &fakeChannel{noConfirm: true}
	p := New(ch)
	require.NoError(t, p.Setup())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, p.Publish(ctx, "event", RoutingKey, nil), context.Canceled)
}

func TestPublishRejectsMandatoryReturnBeforeConfirm(t *testing.T) {
	ch := &fakeChannel{returned: true, noConfirm: true}
	p := New(ch)
	require.NoError(t, p.Setup())
	require.Error(t, p.PublishTaskCreated(context.Background(), 1, "x"))
}

func TestPublishRejectsMandatoryReturnAfterConfirm(t *testing.T) {
	ch := &fakeChannel{returned: true, returnAfterConfirm: true}
	p := New(ch)
	require.NoError(t, p.Setup())
	require.Error(t, p.PublishTaskCreated(context.Background(), 1, "x"))
}
