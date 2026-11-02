import { HttpErrorResponse } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../core/services/auth.service';
import { LoginComponent } from './login.component';

describe('LoginComponent', () => {
  let fixture: ComponentFixture<LoginComponent>;
  const auth = { login: vi.fn(() => of({})) };

  beforeEach(async () => {
    auth.login.mockReset();
    auth.login.mockReturnValue(of({}));
    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [provideRouter([]), { provide: AuthService, useValue: auth }],
    }).compileComponents();
    fixture = TestBed.createComponent(LoginComponent);
    fixture.detectChanges();
  });

  it('blocks submission and exposes accessible errors when invalid', () => {
    fixture.componentInstance.submit();
    fixture.detectChanges();

    expect(auth.login).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('[role="alert"]')).not.toBeNull();
  });

  it('reports invalid credentials for a 401 response', () => {
    auth.login.mockReturnValue(
      throwError(() => new HttpErrorResponse({ status: 401, statusText: 'Unauthorized' })),
    );
    fixture.componentInstance.email = 'user@example.com';
    fixture.componentInstance.password = 'secret123';

    fixture.componentInstance.submit();
    fixture.detectChanges();

    expect(fixture.componentInstance.error()).toBe('Credenciais inválidas.');
  });

  it('reports rate limiting for a 429 response', () => {
    auth.login.mockReturnValue(
      throwError(() => new HttpErrorResponse({ status: 429, statusText: 'Too Many Requests' })),
    );
    fixture.componentInstance.email = 'user@example.com';
    fixture.componentInstance.password = 'secret123';

    fixture.componentInstance.submit();

    expect(fixture.componentInstance.error()).toContain('Muitas tentativas');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('binds fields, clears an error and submits through the form', async () => {
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    fixture.componentInstance.error.set('Erro anterior');
    const root = fixture.nativeElement as HTMLElement;
    const email = root.querySelector('#login-email') as HTMLInputElement;
    const password = root.querySelector('#login-password') as HTMLInputElement;

    email.value = 'user@example.com';
    email.dispatchEvent(new Event('input'));
    password.value = 'secret123';
    password.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    (root.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit'));

    expect(fixture.componentInstance.error()).toBe('');
    expect(auth.login).toHaveBeenCalledWith({ email: 'user@example.com', password: 'secret123' });
    expect(navigate).toHaveBeenCalledWith('/dashboard');
  });
});
