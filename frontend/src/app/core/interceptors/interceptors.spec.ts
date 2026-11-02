import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { jwtInterceptor } from './jwt.interceptor';

describe('HTTP interceptors', () => {
  const auth = { sessionToken: () => 'valid-token', clearSession: vi.fn() };

  beforeEach(() => {
    auth.clearSession.mockClear();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([jwtInterceptor])),
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

  it('adds JWT to the identity profile routes but not to login or register', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);

    for (const url of ['/api/identity/me', '/api/identity/me/password']) {
      http.get(url).subscribe();
      expect(requests.expectOne(url).request.headers.get('Authorization')).toBe('Bearer valid-token');
    }
    for (const url of ['/api/identity/login', '/api/identity/register', '/api/identity/meta']) {
      http.post(url, {}).subscribe();
      expect(requests.expectOne(url).request.headers.has('Authorization')).toBe(false);
    }
  });

  it('clears session after the profile API returns 401', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);
    vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);

    http.get('/api/identity/me').subscribe({ error: () => undefined });
    requests.expectOne('/api/identity/me').flush({}, { status: 401, statusText: 'Unauthorized' });

    expect(auth.clearSession).toHaveBeenCalledOnce();
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

  it('leaves trace and correlation IDs for the edge proxy', () => {
    const http = TestBed.inject(HttpClient);
    const requests = TestBed.inject(HttpTestingController);

    http.get('/api/identity/health').subscribe();
    const request = requests.expectOne('/api/identity/health').request;

    expect(request.headers.has('x-correlation-id')).toBe(false);
    expect(request.headers.has('traceparent')).toBe(false);
  });
});
