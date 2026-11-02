import { HttpErrorResponse } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthService } from '../../../core/services/auth.service';
import { RegisterComponent } from './register.component';

describe('RegisterComponent', () => {
  let fixture: ComponentFixture<RegisterComponent>;
  const auth = { register: vi.fn(() => of({})) };

  beforeEach(async () => {
    auth.register.mockReset();
    auth.register.mockReturnValue(of({}));
    await TestBed.configureTestingModule({
      imports: [RegisterComponent],
      providers: [provideRouter([]), { provide: AuthService, useValue: auth }],
    }).compileComponents();
    fixture = TestBed.createComponent(RegisterComponent);
    fixture.detectChanges();
  });

  it('blocks an invalid email and short password with an accessible error', () => {
    fixture.componentInstance.email = 'invalid';
    fixture.componentInstance.password = '123';

    fixture.componentInstance.submit();
    fixture.detectChanges();

    expect(auth.register).not.toHaveBeenCalled();
    expect(fixture.nativeElement.querySelector('[role="alert"]')?.textContent).toContain(
      'Revise os campos',
    );
  });

  it('reports an existing e-mail for a 409 response', () => {
    auth.register.mockReturnValue(
      throwError(() => new HttpErrorResponse({ status: 409, statusText: 'Conflict' })),
    );
    fixture.componentInstance.email = 'user@example.com';
    fixture.componentInstance.password = 'secret123';

    fixture.componentInstance.submit();

    expect(fixture.componentInstance.error()).toBe('E-mail já cadastrado.');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('reports rate limiting for a 429 response', () => {
    auth.register.mockReturnValue(
      throwError(() => new HttpErrorResponse({ status: 429, statusText: 'Too Many Requests' })),
    );
    fixture.componentInstance.email = 'user@example.com';
    fixture.componentInstance.password = 'secret123';

    fixture.componentInstance.submit();

    expect(fixture.componentInstance.error()).toContain('Muitas tentativas');
    expect(fixture.componentInstance.loading()).toBe(false);
  });
});
