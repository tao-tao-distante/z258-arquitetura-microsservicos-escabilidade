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
	"go.opentelemetry.io/otel/codes"
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
}

// NewDispatcher cria o dispatcher com store, publisher e logger informados.
func NewDispatcher(store OutboxStore, publisher ReliablePublisher, log *slog.Logger) *Dispatcher {
	return &Dispatcher{store: store, publisher: publisher, log: log}
}

// Run despacha um evento a cada 500 ms até o contexto ser cancelado.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := d.DispatchOne(ctx); err != nil && ctx.Err() == nil {
			d.log.ErrorContext(ctx, "outbox dispatch failed", "error", err, "correlation_id", "")
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
	ctx, span := otel.Tracer("service-task/messaging").Start(ctx, "outbox dispatch")
	defer span.End()
	event, err := d.store.ClaimOutbox(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("claim event: %w", err)
	}
	if event == nil {
		return nil
	}
	span.SetAttributes(attribute.String("messaging.message.id", event.EventID))
	eventCtx := correlation.NewContext(ctx, event.CorrelationID)
	if err := d.publisher.Publish(eventCtx, event.EventID, event.EventType, event.Payload); err != nil {
		platformlogger.WithCorrelation(eventCtx, d.log).ErrorContext(eventCtx, "outbox publish failed", "error", err, "event_id", event.EventID)
		if retryErr := d.store.RetryOutbox(eventCtx, event.ID, event.Attempts); retryErr != nil {
			return fmt.Errorf("schedule retry: %w", retryErr)
		}
		return nil
	}
	if err := d.store.MarkOutboxSent(eventCtx, event.ID); err != nil {
		return fmt.Errorf("mark event sent: %w", err)
	}
	return nil
}
