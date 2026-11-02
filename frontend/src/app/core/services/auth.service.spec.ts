import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { AuthResponse } from '../models/auth.model';
import { AuthService } from './auth.service';

type IsOptional<T, K extends keyof T> = Record<string, never> extends Pick<T, K> ? true : false;
const authResponseNameIsRequired: IsOptional<AuthResponse, 'name'> = false;

function token(exp: number): string {
  const payload = btoa(JSON.stringify({ exp })).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
  return `header.${payload}.signature`;
}

describe('AuthService', () => {
  it('requires the user name in authentication responses', () => {
    expect(authResponseNameIsRequired).toBe(false);
  });

  beforeEach(() => {
    window.localStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
  });

  it('removes an expired JWT and its user from storage', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) - 1));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));

    const service = TestBed.inject(AuthService);

    expect(service.token()).toBeNull();
    expect(window.localStorage.getItem('tm_token')).toBeNull();
    expect(window.localStorage.getItem('tm_user')).toBeNull();
  });

  it('removes a malformed JWT', () => {
    window.localStorage.setItem('tm_token', 'not-a-jwt');

    expect(TestBed.inject(AuthService).token()).toBeNull();
    expect(window.localStorage.getItem('tm_token')).toBeNull();
  });

  it('accepts a non-expired JWT', () => {
    const validToken = token(Math.floor(Date.now() / 1000) + 60);
    window.localStorage.setItem('tm_token', validToken);
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));

    expect(TestBed.inject(AuthService).token()).toBe(validToken);
  });

  it('clears a JWT that expires while the application is open', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) + 60));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));
    const service = TestBed.inject(AuthService);
    service.token.set(token(Math.floor(Date.now() / 1000) - 1));

    expect(service.hasValidSession()).toBe(false);
    expect(window.localStorage.getItem('tm_token')).toBeNull();
  });

  it('restores a stored user saved before the name field existed', () => {
    window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) + 60));
    window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com' }));

    expect(TestBed.inject(AuthService).user()).toEqual({ id: 1, email: 'user@example.com', name: '' });
  });

  it('keeps the name returned by login', () => {
    const service = TestBed.inject(AuthService);
    const http = TestBed.inject(HttpTestingController);

    service.login({ email: 'user@example.com', password: 'secret123' }).subscribe();
    http.expectOne('/api/identity/login').flush({
      id: 1,
      email: 'user@example.com',
      name: 'Ana',
      token: token(Math.floor(Date.now() / 1000) + 60),
    });

    expect(service.user()?.name).toBe('Ana');
    expect(JSON.parse(window.localStorage.getItem('tm_user') ?? '{}').name).toBe('Ana');
  });

  it('persists the user returned by registration', () => {
    const service = TestBed.inject(AuthService);
    const http = TestBed.inject(HttpTestingController);

    service.register({ email: 'new@example.com', password: 'secret123' }).subscribe();
    const request = http.expectOne('/api/identity/register');
    expect(request.request.method).toBe('POST');
    request.flush({
      id: 2,
      email: 'new@example.com',
      name: 'Nova',
      token: token(Math.floor(Date.now() / 1000) + 60),
    });

    expect(service.user()).toEqual({ id: 2, email: 'new@example.com', name: 'Nova' });
  });

  it('clears the session and redirects on logout', () => {
    const service = TestBed.inject(AuthService);
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    service.token.set(token(Math.floor(Date.now() / 1000) + 60));
    service.user.set({ id: 1, email: 'user@example.com', name: 'Ana' });

    service.logout();

    expect(service.hasValidSession()).toBe(false);
    expect(navigate).toHaveBeenCalledWith(['/login']);
  });

  describe('profile', () => {
    let service: AuthService;
    let http: HttpTestingController;

    beforeEach(() => {
      window.localStorage.setItem('tm_token', token(Math.floor(Date.now() / 1000) + 60));
      window.localStorage.setItem('tm_user', JSON.stringify({ id: 1, email: 'user@example.com', name: '' }));
      service = TestBed.inject(AuthService);
      http = TestBed.inject(HttpTestingController);
    });

    afterEach(() => http.verify());

    it('loads the profile and refreshes the stored user', () => {
      service.loadProfile().subscribe();
      const req = http.expectOne('/api/identity/me');
      expect(req.request.method).toBe('GET');
      req.flush({ id: 1, email: 'user@example.com', name: 'Ana' });

      expect(service.user()).toEqual({ id: 1, email: 'user@example.com', name: 'Ana' });
      expect(JSON.parse(window.localStorage.getItem('tm_user') ?? '{}').name).toBe('Ana');
    });

    it('updates the name with PATCH /me', () => {
      service.updateName('Bia').subscribe();
      const req = http.expectOne('/api/identity/me');
      expect(req.request.method).toBe('PATCH');
      expect(req.request.body).toEqual({ name: 'Bia' });
      req.flush({ id: 1, email: 'user@example.com', name: 'Bia' });

      expect(service.user()?.name).toBe('Bia');
    });

    it('changes the password with PUT /me/password using the API field names', () => {
      let done = false;
      service
        .changePassword({ currentPassword: 'secret123', newPassword: 'new-secret' })
        .subscribe(() => (done = true));
      const req = http.expectOne('/api/identity/me/password');
      expect(req.request.method).toBe('PUT');
      expect(req.request.body).toEqual({ current_password: 'secret123', new_password: 'new-secret' });
      req.flush(null, { status: 204, statusText: 'No Content' });

      expect(done).toBe(true);
    });
  });
});
