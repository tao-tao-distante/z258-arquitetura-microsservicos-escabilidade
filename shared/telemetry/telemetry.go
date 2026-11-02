package telemetry

import (
	"context"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	apitrace "go.opentelemetry.io/otel/trace"
)

// Shutdown encerra o provider de traces e libera recursos.
type Shutdown func(context.Context) error

type exporterFactory func(context.Context, string) (sdktrace.SpanExporter, error)

// Setup configura o provider OTLP e devolve Shutdown, se a exportação está ativa e erro.
func Setup(ctx context.Context, endpoint, serviceName string) (Shutdown, bool, error) {
	return setup(ctx, endpoint, serviceName, func(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
		return otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpoint))
	})
}

func setup(
	ctx context.Context,
	endpoint, serviceName string,
	newExporter exporterFactory,
) (Shutdown, bool, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	if endpoint == "" {
		otel.SetTracerProvider(apitrace.NewNoopTracerProvider())
		return func(context.Context) error { return nil }, false, nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, false, fmt.Errorf("invalid OTLP endpoint %q", endpoint)
	}
	exporter, err := newExporter(ctx, endpoint)
	if err != nil {
		return nil, false, err
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, true, nil
}
