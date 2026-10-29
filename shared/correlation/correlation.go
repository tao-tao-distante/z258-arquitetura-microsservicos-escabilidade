package correlation

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const key ctxKey = "correlation_id"

// FromContext devolve o correlation ID do contexto, ou string vazia.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(key).(string)
	return id
}

// NewContext devolve um contexto com o correlation ID informado.
func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

// NewID gera um correlation ID UUID v4.
func NewID() string {
	return uuid.New().String()
}
