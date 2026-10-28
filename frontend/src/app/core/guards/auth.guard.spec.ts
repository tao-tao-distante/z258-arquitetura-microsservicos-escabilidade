import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { authGuard } from './auth.guard';
import { AuthService } from '../services/auth.service';

describe('authGuard', () => {
  it('preserves an internal returnUrl for unauthenticated users', () => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: { hasValidSession: () => false } },
      ],
    });

    const result = TestBed.runInInjectionContext(() =>
      authGuard({} as never, { url: '/tasks?filter=todo' } as never),
    );

    expect(TestBed.inject(Router).serializeUrl(result as never)).toBe(
      '/login?returnUrl=%2Ftasks%3Ffilter%3Dtodo',
    );
  });
});
