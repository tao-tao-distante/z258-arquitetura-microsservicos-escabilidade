import { HttpErrorResponse, HttpHeaders } from '@angular/common/http';
import { apiErrorMessage, rateLimitMessage } from './http-error.utils';

function httpError(status: number, retryAfter?: string): HttpErrorResponse {
  return new HttpErrorResponse({
    status,
    headers: retryAfter ? new HttpHeaders({ 'Retry-After': retryAfter }) : undefined,
  });
}

describe('HTTP error messages', () => {
  it('uses Retry-After when the API rate limits a request', () => {
    expect(rateLimitMessage(httpError(429, '120'))).toBe(
      'Muitas tentativas. Tente novamente em 120 s.',
    );
  });

  it('falls back to 60 seconds when Retry-After is missing or invalid', () => {
    expect(rateLimitMessage(httpError(429))).toBe(
      'Muitas tentativas. Tente novamente em 60 s.',
    );
    expect(rateLimitMessage(httpError(429, 'invalid'))).toBe(
      'Muitas tentativas. Tente novamente em 60 s.',
    );
  });

  it('maps expired sessions, oversized payloads and rate limits', () => {
    expect(apiErrorMessage(httpError(401), 'fallback')).toBe(
      'Sessão expirada. Faça login novamente.',
    );
    expect(apiErrorMessage(httpError(413), 'fallback')).toBe(
      'O conteúdo enviado excede o limite permitido.',
    );
    expect(apiErrorMessage(httpError(429, '30'), 'fallback')).toBe(
      'Muitas tentativas. Tente novamente em 30 s.',
    );
  });

  it('keeps the caller fallback for unclassified errors', () => {
    expect(apiErrorMessage(httpError(500), 'Falha ao processar.')).toBe(
      'Falha ao processar.',
    );
  });
});
