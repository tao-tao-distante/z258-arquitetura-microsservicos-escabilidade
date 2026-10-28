package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/broker"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/httpx"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const PrefetchCount = 10

type Channel interface {
	broker.Channel
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

type Consumer struct {
	ch     Channel
	log    *slog.Logger
	active atomic.Bool
}

func NewConsumer(ch Channel, log *slog.Logger) *Consumer {
	return &Consumer{ch: ch, log: log}
}

func (c *Consumer) Start(ctx context.Context) error {
	if err := broker.DeclareTopology(c.ch); err != nil {
		return fmt.Errorf("declare topology: %w", err)
	}
	if err := c.ch.Qos(PrefetchCount, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}
	deliveries, err := c.ch.Consume(broker.Queue, "service-notification", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	c.active.Store(true)
	defer c.active.Store(false)
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			if err := c.handle(delivery); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) Active() bool {
	return c.active.Load()
}

func (c *Consumer) handle(delivery amqp.Delivery) (err error) {
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), amqpHeaderCarrier(delivery.Headers))
	ctx, span := otel.Tracer("service-notification/consumer").Start(ctx, "task.created consume",
		trace.WithSpanKind(trace.SpanKindConsumer))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	cid, _ := delivery.Headers[broker.CorrelationIDHeader].(string)
	if _, parseErr := uuid.Parse(cid); parseErr != nil {
		cid = correlation.NewID()
	}
	ctx = correlation.NewContext(ctx, cid)
	log := logger.WithCorrelation(ctx, c.log)

	var event domain.TaskCreatedEvent
	if decodeErr := json.Unmarshal(delivery.Body, &event); decodeErr != nil {
		log.ErrorContext(ctx, "invalid event", "error", decodeErr)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			return fmt.Errorf("nack delivery: %w", nackErr)
		}
		return nil
	}
	if event.TaskID <= 0 || strings.TrimSpace(event.Title) == "" {
		log.ErrorContext(ctx, "invalid event", "error", "task_id must be positive and title must not be empty")
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			return fmt.Errorf("nack delivery: %w", nackErr)
		}
		return nil
	}

	log.InfoContext(ctx, "notification_received",
		"task_id", event.TaskID,
		"title", event.Title,
	)
	if ackErr := delivery.Ack(false); ackErr != nil {
		return fmt.Errorf("ack delivery: %w", ackErr)
	}
	return nil
}

type amqpHeaderCarrier amqp.Table

func (c amqpHeaderCarrier) Get(key string) string {
	for header, value := range c {
		if strings.EqualFold(header, key) {
			text, _ := value.(string)
			return text
		}
	}
	return ""
}

func (c amqpHeaderCarrier) Set(key, value string) { c[key] = value }

func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = amqpHeaderCarrier{}

type Health struct {
	deps map[string]func(context.Context) error
}

func NewHealth(deps map[string]func(context.Context) error) *Health {
	return &Health{deps: deps}
}

func (h *Health) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, nil)
}
