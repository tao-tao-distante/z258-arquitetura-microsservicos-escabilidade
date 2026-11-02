package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type recordingExporter struct {
	spans    []sdktrace.ReadOnlySpan
	shutdown bool
}

func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *recordingExporter) Shutdown(context.Context) error {
	e.shutdown = true
	return nil
}

func TestSetupWithoutEndpointExplicitlyDisablesExport(t *testing.T) {
	shutdown, enabled, err := Setup(context.Background(), "", "service-test")
	require.NoError(t, err)
	require.False(t, enabled)
	require.NoError(t, shutdown(context.Background()))
	fields := otel.GetTextMapPropagator().Fields()
	require.ElementsMatch(t, propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	).Fields(), fields)
}

func TestSetupWithEndpointEnablesExport(t *testing.T) {
	shutdown, enabled, err := Setup(context.Background(), "http://127.0.0.1:4317", "service-test")
	require.NoError(t, err)
	require.True(t, enabled)
	require.NoError(t, shutdown(context.Background()))
}

func TestSetupRejectsInvalidEndpoint(t *testing.T) {
	_, _, err := Setup(context.Background(), "://invalid", "service-test")
	require.Error(t, err)
}

func TestSetupPropagatesExporterError(t *testing.T) {
	_, _, err := setup(context.Background(), "http://collector:4317", "service-test", func(context.Context, string) (sdktrace.SpanExporter, error) {
		return nil, errors.New("exporter failed")
	})
	require.ErrorContains(t, err, "exporter failed")
}

func TestSetupExportsServiceNameAndShutsDown(t *testing.T) {
	oldProvider := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(oldProvider) })

	exporter := &recordingExporter{}
	shutdown, _, err := setup(context.Background(), "http://collector:4317", "service-test", func(context.Context, string) (sdktrace.SpanExporter, error) {
		return exporter, nil
	})
	require.NoError(t, err)

	_, span := otel.Tracer("test").Start(context.Background(), "operation")
	span.End()
	require.NoError(t, shutdown(context.Background()))
	require.True(t, exporter.shutdown)
	require.Len(t, exporter.spans, 1)

	var serviceName string
	for _, attr := range exporter.spans[0].Resource().Attributes() {
		if string(attr.Key) == "service.name" {
			serviceName = attr.Value.AsString()
		}
	}
	require.Equal(t, "service-test", serviceName)
}

func TestDisabledSetupUsesNoopProvider(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	previous := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(previous)
	t.Cleanup(func() { require.NoError(t, previous.Shutdown(context.Background())) })

	_, _, err := Setup(context.Background(), "", "service-test")
	require.NoError(t, err)
	_, span := otel.Tracer("test").Start(context.Background(), "disabled")
	span.End()
	require.Empty(t, exporter.GetSpans())
}
