package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	Exchange      = "tasks"
	Queue         = "task.events"
	RoutingKey    = "task.created"
	DeadLetterXch = "tasks.dlx"
	DeadLetterQ   = "task.events.dlq"
	HeaderCID     = "x-correlation-id"
)

type Channel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	NotifyReturn(returned chan amqp.Return) chan amqp.Return
}

type TaskCreatedEvent struct {
	TaskID int64  `json:"task_id"`
	Title  string `json:"title"`
}

type Publisher struct {
	ch            Channel
	confirmations chan amqp.Confirmation
	returns       chan amqp.Return
	mu            sync.Mutex
}

func New(ch Channel) *Publisher {
	return &Publisher{ch: ch}
}

func (p *Publisher) Setup() error {
	if err := p.ch.ExchangeDeclare(DeadLetterXch, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}
	if _, err := p.ch.QueueDeclare(DeadLetterQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := p.ch.QueueBind(DeadLetterQ, "", DeadLetterXch, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}
	if err := p.ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := p.ch.QueueDeclare(Queue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": DeadLetterXch,
	}); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := p.ch.QueueBind(Queue, RoutingKey, Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	if err := p.ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	p.confirmations = p.ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	p.returns = p.ch.NotifyReturn(make(chan amqp.Return, 1))
	return nil
}

func (p *Publisher) PublishTaskCreated(ctx context.Context, taskID int64, title string) error {
	body, err := json.Marshal(TaskCreatedEvent{TaskID: taskID, Title: title})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.Publish(ctx, fmt.Sprintf("task-%d", taskID), RoutingKey, body)
}

func (p *Publisher) Publish(ctx context.Context, eventID, routingKey string, body []byte) error {
	ctx, span := otel.Tracer("service-task/messaging").Start(ctx, "rabbitmq publish")
	defer span.End()
	span.SetAttributes(
		attribute.String("messaging.system", "rabbitmq"),
		attribute.String("messaging.destination.name", Exchange),
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
	headers := amqp.Table{HeaderCID: correlation.FromContext(ctx)}
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
	if err := p.ch.PublishWithContext(ctx, Exchange, routingKey, true, false, msg); err != nil {
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

func (c tableCarrier) Get(key string) string {
	value, _ := c[key].(string)
	return value
}

func (c tableCarrier) Set(key, value string) { c[key] = value }

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
