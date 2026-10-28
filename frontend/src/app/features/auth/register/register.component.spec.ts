import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../../core/services/auth.service';
import { RegisterComponent } from './register.component';

describe('RegisterComponent', () => {
  let fixture: ComponentFixture<RegisterComponent>;
  const auth = { register: vi.fn(() => of({})) };

  beforeEach(async () => {
    auth.register.mockClear();
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
});
