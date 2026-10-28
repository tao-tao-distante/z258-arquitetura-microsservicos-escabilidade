import { provideHttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from './auth.service';

function token(exp: number): string {
  const payload = btoa(JSON.stringify({ exp })).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
  return `header.${payload}.signature`;
}

describe('AuthService', () => {
  beforeEach(() => {
    window.localStorage.clear();
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideRouter([])] });
  });

  it('removes an expired JWT and its user from storage', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) - 1));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));

    const service = TestBed.inject(AuthService);

    expect(service.isAuthenticated()).toBe(false);
    expect(window.localStorage.getItem('tm_token')).toBeNull();
    expect(window.localStorage.getItem('tm_user')).toBeNull();
  });

  it('removes a malformed JWT', () => {
    window.localStorage.setItem('tm_token', 'not-a-jwt');

    expect(TestBed.inject(AuthService).token()).toBeNull();
    expect(window.localStorage.getItem('tm_token')).toBeNull();
  });

  it('accepts a non-expired JWT', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) + 60));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));

    expect(TestBed.inject(AuthService).isAuthenticated()).toBe(true);
  });

  it('clears a JWT that expires while the application is open', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) + 60));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));
    const service = TestBed.inject(AuthService);
    service.token.set(token(Math.floor(Date.now() / 1000) - 1));

    expect(service.hasValidSession()).toBe(false);
    expect(window.localStorage.getItem('tm_token')).toBeNull();
  });
});
