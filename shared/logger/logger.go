package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

// New cria um logger slog JSON em stdout no nível Info.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// WithCorrelation adiciona o correlation_id do contexto ao logger.
func WithCorrelation(ctx context.Context, l *slog.Logger) *slog.Logger {
	if id := correlation.FromContext(ctx); id != "" {
		return l.With("correlation_id", id)
	}
	return l
}
