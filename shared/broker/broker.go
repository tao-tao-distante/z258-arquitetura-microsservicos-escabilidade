package broker

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Nomes da topologia AMQP compartilhada entre publicadores e consumidores.
const (
	Exchange            = "tasks"
	Queue               = "task.events"
	RoutingKey          = "task.created"
	DeadLetterExchange  = "tasks.dlx"
	DeadLetterQueue     = "task.events.dlq"
	CorrelationIDHeader = "x-correlation-id"
)

// Channel abstrai as operações de declaração necessárias à topologia AMQP.
type Channel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
}

// DeclareTopology cria exchange, fila, DLX, DLQ e bindings de forma idempotente.
func DeclareTopology(ch Channel) error {
	if err := ch.ExchangeDeclare(DeadLetterExchange, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}
	if _, err := ch.QueueDeclare(DeadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(DeadLetterQueue, "", DeadLetterExchange, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}
	if err := ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(Queue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": DeadLetterExchange,
	}); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(Queue, RoutingKey, Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	return nil
}
