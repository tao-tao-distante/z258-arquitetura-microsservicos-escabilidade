package messaging

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/broker"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Channel abstrai o canal AMQP com publisher confirms.
type Channel interface {
	broker.Channel
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	NotifyReturn(returned chan amqp.Return) chan amqp.Return
}

// Publisher publica eventos com confirms e detecção de retorno.
type Publisher struct {
	ch            Channel
	confirmations chan amqp.Confirmation
	returns       chan amqp.Return
	mu            sync.Mutex
}

// New cria o publisher com o canal informado.
func New(ch Channel) *Publisher {
	return &Publisher{ch: ch}
}

// Setup declara a topologia e habilita publisher confirms e returns.
func (p *Publisher) Setup() error {
	if err := broker.DeclareTopology(p.ch); err != nil {
		return err
	}
	if err := p.ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	p.confirmations = p.ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	p.returns = p.ch.NotifyReturn(make(chan amqp.Return, 1))
	return nil
}

// Publish publica um evento persistente e aguarda confirmação de roteamento.
func (p *Publisher) Publish(ctx context.Context, eventID, routingKey string, body []byte) error {
	ctx, span := otel.Tracer("service-task/messaging").Start(ctx, "rabbitmq publish", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", broker.Exchange),
		attribute.String("messaging.message.id", eventID),
		attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
	)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.confirmations == nil || p.returns == nil {
		err := errors.New("publisher is not configured")
		recordSpanError(span, err)
		return err
	}
	headers := amqp.Table{broker.CorrelationIDHeader: correlation.FromContext(ctx)}
	otel.GetTextMapPropagator().Inject(ctx, tableCarrier(headers))
	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Headers:      headers,
		Body:         body,
		MessageId:    eventID,
		Type:         routingKey,
	}
	if err := p.ch.PublishWithContext(ctx, broker.Exchange, routingKey, true, false, msg); err != nil {
		recordSpanError(span, err)
		return fmt.Errorf("publish event: %w", err)
	}
	for {
		select {
		case returned := <-p.returns:
			if returned.MessageId == eventID {
				err := errors.New("published event was unroutable")
				recordSpanError(span, err)
				return err
			}
		case confirmation := <-p.confirmations:
			if !confirmation.Ack {
				err := errors.New("published event was nacked")
				recordSpanError(span, err)
				return err
			}
			// amqp091-go cannot guarantee ordering between return and ack channels.
			timer := time.NewTimer(10 * time.Millisecond)
			select {
			case returned := <-p.returns:
				timer.Stop()
				if returned.MessageId == eventID {
					err := errors.New("published event was unroutable")
					recordSpanError(span, err)
					return err
				}
			case <-timer.C:
			}
			return nil
		case <-ctx.Done():
			recordSpanError(span, ctx.Err())
			return ctx.Err()
		}
	}
}

type tableCarrier amqp.Table

// Get devolve o valor textual da chave no header AMQP.
func (c tableCarrier) Get(key string) string {
	value, _ := c[key].(string)
	return value
}

// Set grava um valor textual no header AMQP.
func (c tableCarrier) Set(key, value string) { c[key] = value }

// Keys lista as chaves presentes no header AMQP.
func (c tableCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

func recordSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
