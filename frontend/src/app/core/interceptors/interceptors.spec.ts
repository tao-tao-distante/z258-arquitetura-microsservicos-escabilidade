import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { correlationIdInterceptor } from './correlation-id.interceptor';
import { jwtInterceptor } from './jwt.interceptor';

describe('HTTP interceptors', () => {
  const auth = { sessionToken: () => 'valid-token', clearSession: vi.fn() };

  beforeEach(() => {
    auth.clearSession.mockClear();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([correlationIdInterceptor, jwtInterceptor])),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: auth },
      ],
    });
  });

  it('adds JWT only to task API requests', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);

    http.get('/api/tasks/tasks').subscribe();
    expect(requests.expectOne('/api/tasks/tasks').request.headers.get('Authorization')).toBe(
      'Bearer valid-token',
    );

    http.get('/api/identity/health').subscribe();
    expect(requests.expectOne('/api/identity/health').request.headers.has('Authorization')).toBe(false);
  });

  it('clears session and redirects once after task API returns 401', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);
    const router = TestBed.inject(Router);
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    http.get('/api/tasks/tasks').subscribe({ error: () => undefined });
    requests.expectOne('/api/tasks/tasks').flush({}, { status: 401, statusText: 'Unauthorized' });

    expect(auth.clearSession).toHaveBeenCalledOnce();
    expect(navigate).toHaveBeenCalledOnce();
  });

  it('always supplies a UUID-shaped correlation ID when randomUUID is unavailable', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);
    const original = crypto.randomUUID;
    Object.defineProperty(crypto, 'randomUUID', { configurable: true, value: undefined });

    http.get('/api/identity/health').subscribe();
    const id = requests.expectOne('/api/identity/health').request.headers.get('x-correlation-id');

    Object.defineProperty(crypto, 'randomUUID', { configurable: true, value: original });
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });
});
