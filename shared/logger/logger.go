package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel/trace"
)

// New cria um logger slog JSON em stdout no nível Info.
func New(serviceName string) *slog.Logger {
	return newLogger(os.Stdout, serviceName)
}

func newLogger(output io.Writer, serviceName string) *slog.Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler).With("service_name", serviceName)
}

// WithContext adiciona os identificadores disponíveis no contexto ao logger.
func WithContext(ctx context.Context, l *slog.Logger) *slog.Logger {
	attrs := make([]any, 0, 6)
	if id := correlation.FromContext(ctx); id != "" {
		attrs = append(attrs, "correlation_id", id)
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		attrs = append(attrs,
			"trace_id", spanContext.TraceID().String(),
			"span_id", spanContext.SpanID().String(),
		)
	}
	return l.With(attrs...)
}
