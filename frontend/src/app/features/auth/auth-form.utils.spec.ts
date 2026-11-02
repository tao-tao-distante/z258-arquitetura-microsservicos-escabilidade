import { isValidEmail, isValidPassword, safeReturnUrl } from './auth-form.utils';

describe('auth form utils', () => {
  it('validates emails', () => {
    expect(isValidEmail(' user@example.com ')).toBe(true);
    expect(isValidEmail('invalid')).toBe(false);
    expect(isValidEmail('a@b')).toBe(false);
  });

  it('enforces password length bounds', () => {
    expect(isValidPassword('1234567')).toBe(false);
    expect(isValidPassword('12345678')).toBe(true);
    expect(isValidPassword('a'.repeat(73))).toBe(false);
  });

  it('only accepts local return URLs', () => {
    expect(safeReturnUrl('/tasks')).toBe('/tasks');
    expect(safeReturnUrl('//evil.test')).toBe('/tasks');
    expect(safeReturnUrl('https://evil.test')).toBe('/tasks');
    expect(safeReturnUrl('/tasks\\evil')).toBe('/tasks');
    expect(safeReturnUrl(null)).toBe('/tasks');
  });
});
