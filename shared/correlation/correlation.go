package correlation

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const key ctxKey = "correlation_id"

func FromContext(ctx context.Context) string {
	if v := ctx.Value(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func NewContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

func NewID() string {
	return uuid.New().String()
}
