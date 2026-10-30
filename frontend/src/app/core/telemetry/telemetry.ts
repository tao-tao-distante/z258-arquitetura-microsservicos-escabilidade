import { propagation, trace } from '@opentelemetry/api';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { BatchSpanProcessor, SpanProcessor, TracerProvider } from '@opentelemetry/sdk-trace';
import { ATTR_SERVICE_NAME } from '@opentelemetry/semantic-conventions';

export function setupTelemetry(
  spanProcessor: SpanProcessor = new BatchSpanProcessor({
    exporter: new OTLPTraceExporter({ url: '/otlp/v1/traces' }),
  }),
): TracerProvider {
  const provider = new TracerProvider({
    resource: resourceFromAttributes({ [ATTR_SERVICE_NAME]: 'frontend' }),
    spanProcessors: [spanProcessor],
  });
  trace.setGlobalTracerProvider(provider);
  propagation.setGlobalPropagator(new W3CTraceContextPropagator());
  return provider;
}
