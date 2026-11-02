import { HttpErrorResponse } from '@angular/common/http';

const DEFAULT_RETRY_AFTER_SECONDS = 60;

export function rateLimitMessage(error: HttpErrorResponse): string {
  const header = error.headers.get('Retry-After');
  const parsed = header === null ? Number.NaN : Number.parseInt(header, 10);
  const seconds = Number.isFinite(parsed) && parsed > 0
    ? parsed
    : DEFAULT_RETRY_AFTER_SECONDS;
  return `Muitas tentativas. Tente novamente em ${seconds} s.`;
}

export function apiErrorMessage(error: HttpErrorResponse, fallback: string): string {
  switch (error.status) {
    case 401:
      return 'Sessão expirada. Faça login novamente.';
    case 413:
      return 'O conteúdo enviado excede o limite permitido.';
    case 429:
      return rateLimitMessage(error);
    default:
      return fallback;
  }
}
