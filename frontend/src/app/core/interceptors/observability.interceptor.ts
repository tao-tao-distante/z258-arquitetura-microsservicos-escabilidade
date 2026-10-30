import { HttpInterceptorFn, HttpResponse } from '@angular/common/http';
import { context, propagation, SpanKind, SpanStatusCode, trace } from '@opentelemetry/api';
import {
  ATTR_HTTP_REQUEST_METHOD,
  ATTR_HTTP_RESPONSE_STATUS_CODE,
  ATTR_URL_PATH,
} from '@opentelemetry/semantic-conventions';
import { finalize, tap } from 'rxjs';

export const observabilityInterceptor: HttpInterceptorFn = (req, next) => {
  const correlationId = createCorrelationId();
  const span = trace.getTracer('frontend/http').startSpan(`HTTP ${req.method}`, {
    kind: SpanKind.CLIENT,
    attributes: {
      [ATTR_HTTP_REQUEST_METHOD]: req.method,
      [ATTR_URL_PATH]: req.url.split('?')[0],
    },
  });
  const traceHeaders: Record<string, string> = {};
  propagation.inject(trace.setSpan(context.active(), span), traceHeaders);

  return next(
    req.clone({ setHeaders: { 'x-correlation-id': correlationId, ...traceHeaders } }),
  ).pipe(
    tap({
      next: (event) => {
        if (event instanceof HttpResponse) {
          span.setAttribute(ATTR_HTTP_RESPONSE_STATUS_CODE, event.status);
        }
      },
      error: (error: unknown) => {
        span.recordException(error instanceof Error ? error : String(error));
        span.setStatus({ code: SpanStatusCode.ERROR });
      },
    }),
    finalize(() => span.end()),
  );
};

function createCorrelationId(): string {
  if (typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
