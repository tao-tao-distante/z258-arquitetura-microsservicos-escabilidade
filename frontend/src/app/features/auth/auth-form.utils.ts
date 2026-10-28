export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_LENGTH = 72;

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export function isValidPassword(password: string, minLength = MIN_PASSWORD_LENGTH): boolean {
  return password.length >= minLength && password.length <= MAX_PASSWORD_LENGTH;
}

export function safeReturnUrl(returnUrl: string | null, fallback = '/tasks'): string {
  return returnUrl?.startsWith('/') && !returnUrl.startsWith('//') && !returnUrl.includes('\\')
    ? returnUrl
    : fallback;
}
