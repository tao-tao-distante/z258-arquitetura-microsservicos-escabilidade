package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	platformlogger "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// OutboxStore define as operações de outbox usadas pelo dispatcher.
type OutboxStore interface {
	ClaimOutbox(ctx context.Context) (*repository.OutboxEvent, error)
	MarkOutboxSent(ctx context.Context, id int64) error
	RetryOutbox(ctx context.Context, id int64, attempts int) error
}

// ReliablePublisher publica eventos com confirmação de entrega.
type ReliablePublisher interface {
	Publish(ctx context.Context, eventID, routingKey string, payload []byte) error
}

// Dispatcher publica eventos pendentes do outbox em intervalos regulares.
type Dispatcher struct {
	store     OutboxStore
	publisher ReliablePublisher
	log       *slog.Logger
	metrics   counter
}

type counter interface{ Inc(...string) }

// NewDispatcher cria o dispatcher com suas dependências.
func NewDispatcher(store OutboxStore, publisher ReliablePublisher, log *slog.Logger, metrics counter) *Dispatcher {
	return &Dispatcher{store: store, publisher: publisher, log: log, metrics: metrics}
}

// Run despacha um evento a cada 500 ms até o contexto ser cancelado.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := d.DispatchOne(ctx); err != nil && ctx.Err() == nil {
			d.log.ErrorContext(ctx, "outbox dispatch failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// DispatchOne reivindica, publica e confirma um único evento do outbox.
func (d *Dispatcher) DispatchOne(ctx context.Context) error {
	event, err := d.store.ClaimOutbox(ctx)
	if err != nil {
		d.metrics.Inc("claim_error")
		return fmt.Errorf("claim event: %w", err)
	}
	if event == nil {
		return nil
	}
	eventCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.TraceContext))
	eventCtx = correlation.NewContext(eventCtx, event.CorrelationID)
	eventCtx, span := otel.Tracer("service-task/messaging").Start(eventCtx, "outbox dispatch")
	defer span.End()
	span.SetAttributes(attribute.String("messaging.message.id", event.EventID))
	if err := d.publisher.Publish(eventCtx, event.EventID, event.EventType, event.Payload); err != nil {
		platformlogger.WithContext(eventCtx, d.log).ErrorContext(eventCtx, "outbox publish failed", "error", err, "event_id", event.EventID)
		if retryErr := d.store.RetryOutbox(eventCtx, event.ID, event.Attempts); retryErr != nil {
			d.metrics.Inc("retry_error")
			return fmt.Errorf("schedule retry: %w", retryErr)
		}
		d.metrics.Inc("retry")
		platformlogger.WithContext(eventCtx, d.log).WarnContext(eventCtx, "outbox_event_retry_scheduled", "event_id", event.EventID, "attempts", event.Attempts)
		return nil
	}
	if err := d.store.MarkOutboxSent(eventCtx, event.ID); err != nil {
		d.metrics.Inc("mark_error")
		return fmt.Errorf("mark event sent: %w", err)
	}
	platformlogger.WithContext(eventCtx, d.log).InfoContext(eventCtx, "outbox_event_published", "event_id", event.EventID, "event_type", event.EventType)
	d.metrics.Inc("sent")
	return nil
}
