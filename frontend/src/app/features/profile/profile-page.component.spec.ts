import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { AuthUser } from '../../core/models/auth.model';
import { AuthService } from '../../core/services/auth.service';
import { ProfilePageComponent } from './profile-page.component';

describe('ProfilePageComponent', () => {
  let fixture: ComponentFixture<ProfilePageComponent>;
  const profile: AuthUser = { id: 1, email: 'user@example.com', name: 'Ana' };
  const auth = {
    user: signal<AuthUser | null>({ id: 1, email: 'user@example.com', name: '' }),
    loadProfile: vi.fn(() => of(profile)),
    updateName: vi.fn((name: string) => of({ ...profile, name })),
    changePassword: vi.fn(() => of(undefined)),
  };

  function httpError(status: number): HttpErrorResponse {
    return new HttpErrorResponse({ status, statusText: 'Error' });
  }

  function fillPasswords(current: string, next: string, confirmation = next): void {
    fixture.componentInstance.currentPassword = current;
    fixture.componentInstance.newPassword = next;
    fixture.componentInstance.confirmPassword = confirmation;
  }

  beforeEach(async () => {
    auth.loadProfile.mockClear();
    auth.updateName.mockClear();
    auth.changePassword.mockReset();
    auth.changePassword.mockReturnValue(of(undefined));
    await TestBed.configureTestingModule({
      imports: [ProfilePageComponent],
      providers: [provideRouter([]), { provide: AuthService, useValue: auth }],
    }).compileComponents();
    fixture = TestBed.createComponent(ProfilePageComponent);
    fixture.detectChanges();
  });

  it('loads the profile and fills the name field', () => {
    expect(auth.loadProfile).toHaveBeenCalledOnce();
    expect(fixture.componentInstance.name).toBe('Ana');
    expect(fixture.nativeElement.querySelector('#profile-email')?.value).toBe('user@example.com');
  });

  it('reports a failure to load the profile', () => {
    auth.loadProfile.mockReturnValueOnce(throwError(() => httpError(500)));
    fixture = TestBed.createComponent(ProfilePageComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.profileError()).toBe('Não foi possível carregar o perfil.');
  });

  describe('name', () => {
    it('blocks a blank name with an accessible error', () => {
      fixture.componentInstance.name = '   ';

      fixture.componentInstance.saveName();
      fixture.detectChanges();

      expect(auth.updateName).not.toHaveBeenCalled();
      expect(fixture.nativeElement.querySelector('[role="alert"]')?.textContent).toContain('Informe seu nome');
    });

    it('saves the trimmed name and confirms the change', () => {
      fixture.componentInstance.name = '  Bia Lima ';

      fixture.componentInstance.saveName();
      fixture.detectChanges();

      expect(auth.updateName).toHaveBeenCalledWith('Bia Lima');
      expect(fixture.componentInstance.name).toBe('Bia Lima');
      expect(fixture.nativeElement.querySelector('[role="status"]')?.textContent).toContain('Nome atualizado');
      expect(fixture.componentInstance.savingName()).toBe(false);
    });

    it('reports an API failure', () => {
      auth.updateName.mockReturnValueOnce(throwError(() => httpError(500)));
      fixture.componentInstance.name = 'Bia';

      fixture.componentInstance.saveName();

      expect(fixture.componentInstance.profileError()).toBe('Falha ao salvar o nome.');
      expect(fixture.componentInstance.savingName()).toBe(false);
    });
  });

  describe('password', () => {
    it.each([
      ['a short new password', 'secret123', 'short', 'short', 'A nova senha deve ter de 8 a 72 caracteres.'],
      ['a confirmation mismatch', 'secret123', 'new-secret', 'other-secret', 'A confirmação não confere com a nova senha.'],
      ['the same password', 'secret123', 'secret123', 'secret123', 'A nova senha deve ser diferente da atual.'],
      ['a missing current password', '', 'new-secret', 'new-secret', 'Informe a senha atual.'],
    ])('blocks %s', (_case, current, next, confirmation, message) => {
      fillPasswords(current, next, confirmation);

      fixture.componentInstance.changePassword();

      expect(auth.changePassword).not.toHaveBeenCalled();
      expect(fixture.componentInstance.passwordError()).toBe(message);
    });

    it('changes the password, clears the fields and confirms', () => {
      fillPasswords('secret123', 'new-secret');

      fixture.componentInstance.changePassword();
      fixture.detectChanges();

      expect(auth.changePassword).toHaveBeenCalledWith({
        currentPassword: 'secret123',
        newPassword: 'new-secret',
      });
      expect(fixture.componentInstance.currentPassword).toBe('');
      expect(fixture.componentInstance.newPassword).toBe('');
      expect(fixture.componentInstance.confirmPassword).toBe('');
      expect(fixture.componentInstance.passwordSuccess()).toBe('Senha alterada com sucesso.');
    });

    it.each([
      [403, 'Senha atual incorreta.'],
      [429, 'Muitas tentativas'],
      [500, 'Falha ao alterar a senha.'],
    ])('reports a %i response', (status, message) => {
      auth.changePassword.mockReturnValueOnce(throwError(() => httpError(status)));
      fillPasswords('secret123', 'new-secret');

      fixture.componentInstance.changePassword();

      expect(fixture.componentInstance.passwordError()).toContain(message);
      expect(fixture.componentInstance.changingPassword()).toBe(false);
    });
  });

  it('binds and submits both profile forms through the DOM', async () => {
    const root = fixture.nativeElement as HTMLElement;
    const change = async (selector: string, value: string) => {
      const input = root.querySelector(selector) as HTMLInputElement;
      input.value = value;
      input.dispatchEvent(new Event('input'));
      await fixture.whenStable();
    };

    fixture.componentInstance.profileError.set('Erro anterior');
    await change('#profile-name', 'Bia');
    (root.querySelector('[aria-labelledby="profile-data-title"]') as HTMLFormElement)
      .dispatchEvent(new Event('submit'));

    fixture.componentInstance.passwordError.set('Erro anterior');
    await change('#profile-current-password', 'secret123');
    await change('#profile-new-password', 'new-secret');
    await change('#profile-confirm-password', 'new-secret');
    (root.querySelector('[aria-labelledby="profile-password-title"]') as HTMLFormElement)
      .dispatchEvent(new Event('submit'));

    expect(auth.updateName).toHaveBeenCalledWith('Bia');
    expect(auth.changePassword).toHaveBeenCalledWith({
      currentPassword: 'secret123',
      newPassword: 'new-secret',
    });
  });
});
