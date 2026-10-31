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

// PrefetchCount limita mensagens não confirmadas entregues ao consumidor.
const PrefetchCount = 10

// Channel abstrai o canal AMQP necessário ao consumo.
type Channel interface {
	broker.Channel
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
}

// Consumer consome eventos task.created da fila de notificações.
type Consumer struct {
	ch      Channel
	log     *slog.Logger
	active  atomic.Bool
	metrics counter
}

type counter interface{ Inc(...string) }

type noopCounter struct{}

func (noopCounter) Inc(...string) {}

// NewConsumer cria o consumidor com o canal e o logger informados.
func NewConsumer(ch Channel, log *slog.Logger, counters ...counter) *Consumer {
	metrics := counter(noopCounter{})
	if len(counters) > 0 && counters[0] != nil {
		metrics = counters[0]
	}
	return &Consumer{ch: ch, log: log, metrics: metrics}
}

// Start declara a topologia, aplica prefetch e consome até o contexto ser cancelado.
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

// Active informa se o consumo está em execução.
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
	log := logger.WithContext(ctx, c.log)

	var event domain.TaskCreatedEvent
	var invalidReason any
	if decodeErr := json.Unmarshal(delivery.Body, &event); decodeErr != nil {
		invalidReason = decodeErr
	} else if event.TaskID <= 0 || strings.TrimSpace(event.Title) == "" {
		invalidReason = "task_id must be positive and title must not be empty"
	}
	if invalidReason != nil {
		log.ErrorContext(ctx, "invalid event", "error", invalidReason)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			c.metrics.Inc("nack_error")
			return fmt.Errorf("nack delivery: %w", nackErr)
		}
		c.metrics.Inc("rejected")
		return nil
	}

	log.InfoContext(ctx, "notification_received", "task_id", event.TaskID)
	if ackErr := delivery.Ack(false); ackErr != nil {
		c.metrics.Inc("ack_error")
		return fmt.Errorf("ack delivery: %w", ackErr)
	}
	c.metrics.Inc("acked")
	return nil
}

type amqpHeaderCarrier amqp.Table

// Get busca a chave no header AMQP sem diferenciar maiúsculas de minúsculas.
func (c amqpHeaderCarrier) Get(key string) string {
	for header, value := range c {
		if strings.EqualFold(header, key) {
			text, _ := value.(string)
			return text
		}
	}
	return ""
}

// Set grava um valor textual no header AMQP.
func (c amqpHeaderCarrier) Set(key, value string) { c[key] = value }

// Keys lista as chaves presentes no header AMQP.
func (c amqpHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = amqpHeaderCarrier{}

// Health expõe o endpoint de saúde do service-notification.
type Health struct {
	deps map[string]func(context.Context) error
}

// NewHealth cria o handler de saúde com as checagens de dependência.
func NewHealth(deps map[string]func(context.Context) error) *Health {
	return &Health{deps: deps}
}

// Health responde o estado do processo e do RabbitMQ.
func (h *Health) Health(w http.ResponseWriter, r *http.Request) {
	httpx.Health(w, r, h.deps, nil)
}
