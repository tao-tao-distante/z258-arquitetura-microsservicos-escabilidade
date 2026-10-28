package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func WithCorrelation(ctx context.Context, l *slog.Logger) *slog.Logger {
	if id := correlation.FromContext(ctx); id != "" {
		return l.With("correlation_id", id)
	}
	return l
}
