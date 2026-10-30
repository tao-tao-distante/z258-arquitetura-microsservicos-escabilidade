import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { finalize } from 'rxjs';
import { apiErrorMessage } from '../../core/http/http-error.utils';
import { AuthService } from '../../core/services/auth.service';
import { FormFieldComponent } from '../../shared/components/form-field.component';
import { PageHeaderComponent } from '../../shared/components/page-header.component';
import { isValidPassword, MAX_PASSWORD_LENGTH, MIN_PASSWORD_LENGTH } from '../auth/auth-form.utils';

export const MAX_NAME_LENGTH = 100;

@Component({
  selector: 'app-profile-page',
  imports: [FormsModule, FormFieldComponent, PageHeaderComponent],
  template: `
    <section class="page" animate.enter="page-enter">
      <app-page-header
        title="Meu perfil."
        subtitle="Como você aparece no workspace e a senha de acesso."
        mobileEyebrow="CONTA"
      />

      <hr class="rule" />

      <form class="form" (ngSubmit)="saveName()" novalidate aria-labelledby="profile-data-title">
        <h2 id="profile-data-title" class="eyebrow">01 / DADOS PESSOAIS</h2>

        <app-form-field label="E-MAIL" inputId="profile-email">
          <input
            id="profile-email"
            class="input"
            name="email"
            type="email"
            [value]="email()"
            readonly
            aria-describedby="profile-email-hint"
          />
          <span id="profile-email-hint" class="hint">O e-mail identifica sua conta e não pode ser alterado.</span>
        </app-form-field>

        <app-form-field label="NOME" inputId="profile-name">
          <input
            id="profile-name"
            class="input"
            name="name"
            [(ngModel)]="name"
            (ngModelChange)="clearProfileMessages()"
            required
            [maxlength]="maxNameLength"
            autocomplete="name"
            placeholder="Como devemos chamar você?"
            aria-describedby="profile-name-feedback"
          />
        </app-form-field>

        <div id="profile-name-feedback">
          @if (profileError()) {
            <p class="error" role="alert" aria-live="assertive">{{ profileError() }}</p>
          }
          @if (profileSuccess()) {
            <p class="success" role="status">{{ profileSuccess() }}</p>
          }
        </div>

        <div class="actions">
          <button class="btn btn-primary" type="submit" [disabled]="savingName()">
            {{ savingName() ? 'Salvando...' : 'Salvar nome' }}
            <span class="arrow" aria-hidden="true">↗</span>
          </button>
        </div>
      </form>

      <hr class="rule section-rule" />

      <form class="form" (ngSubmit)="changePassword()" novalidate aria-labelledby="profile-password-title">
        <h2 id="profile-password-title" class="eyebrow">02 / ALTERAR SENHA</h2>

        <app-form-field label="SENHA ATUAL" inputId="profile-current-password">
          <input
            id="profile-current-password"
            class="input"
            name="currentPassword"
            type="password"
            [(ngModel)]="currentPassword"
            (ngModelChange)="clearPasswordMessages()"
            required
            [maxlength]="maxPasswordLength"
            autocomplete="current-password"
            aria-describedby="profile-password-feedback"
          />
        </app-form-field>

        <app-form-field label="NOVA SENHA" inputId="profile-new-password">
          <input
            id="profile-new-password"
            class="input"
            name="newPassword"
            type="password"
            [(ngModel)]="newPassword"
            (ngModelChange)="clearPasswordMessages()"
            required
            [minlength]="minPasswordLength"
            [maxlength]="maxPasswordLength"
            autocomplete="new-password"
            placeholder="Pelo menos 8 caracteres"
            aria-describedby="profile-password-feedback"
          />
        </app-form-field>

        <app-form-field label="CONFIRMAR NOVA SENHA" inputId="profile-confirm-password">
          <input
            id="profile-confirm-password"
            class="input"
            name="confirmPassword"
            type="password"
            [(ngModel)]="confirmPassword"
            (ngModelChange)="clearPasswordMessages()"
            required
            [maxlength]="maxPasswordLength"
            autocomplete="new-password"
            aria-describedby="profile-password-feedback"
          />
        </app-form-field>

        <div id="profile-password-feedback">
          @if (passwordError()) {
            <p class="error" role="alert" aria-live="assertive">{{ passwordError() }}</p>
          }
          @if (passwordSuccess()) {
            <p class="success" role="status">{{ passwordSuccess() }}</p>
          }
        </div>

        <div class="actions">
          <button class="btn btn-primary" type="submit" [disabled]="changingPassword()">
            {{ changingPassword() ? 'Alterando...' : 'Alterar senha' }}
            <span class="arrow" aria-hidden="true">↗</span>
          </button>
        </div>
      </form>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      app-page-header { margin-bottom: 40px; }
      .rule { margin-bottom: 32px; }
      .section-rule { margin-top: 48px; }
      .form { display: grid; gap: 0; max-width: 720px; }
      .form h2 { margin-bottom: 28px; }
      app-form-field { --field-gap: 14px; }
      app-form-field + app-form-field { margin-top: 32px; }
      input.input { min-height: 60px; }
      input.input[readonly] { background: var(--band-todo); color: var(--muted); }
      .hint { font-size: 12px; color: var(--muted); }
      #profile-name-feedback, #profile-password-feedback { margin-top: 20px; min-height: 1.25rem; }
      .success { color: var(--text-done); font-size: 0.875rem; }
      .actions { display: flex; justify-content: flex-end; margin-top: 16px; }
      .actions .btn-primary { min-width: 224px; }
      @media (max-width: 900px) {
        .form { max-width: none; }
        .input, input.input { min-height: 53px; padding: 0 14px; font-size: 14px; }
        .actions { display: block; margin-top: 24px; }
        .actions .btn-primary { width: 100%; }
      }
    `,
  ],
})
export class ProfilePageComponent implements OnInit {
  private readonly auth = inject(AuthService);

  readonly maxNameLength = MAX_NAME_LENGTH;
  readonly minPasswordLength = MIN_PASSWORD_LENGTH;
  readonly maxPasswordLength = MAX_PASSWORD_LENGTH;
  readonly email = computed(() => this.auth.user()?.email ?? '');

  readonly savingName = signal(false);
  readonly profileError = signal('');
  readonly profileSuccess = signal('');
  readonly changingPassword = signal(false);
  readonly passwordError = signal('');
  readonly passwordSuccess = signal('');

  name = this.auth.user()?.name ?? '';
  currentPassword = '';
  newPassword = '';
  confirmPassword = '';

  ngOnInit(): void {
    this.auth.loadProfile().subscribe({
      next: (user) => (this.name = user.name),
      error: (err: HttpErrorResponse) =>
        this.profileError.set(apiErrorMessage(err, 'Não foi possível carregar o perfil.')),
    });
  }

  saveName(): void {
    const name = this.name.trim();
    if (name.length === 0 || [...name].length > MAX_NAME_LENGTH) {
      this.profileError.set(`Informe seu nome (até ${MAX_NAME_LENGTH} caracteres).`);
      return;
    }
    this.clearProfileMessages();
    this.savingName.set(true);
    this.auth
      .updateName(name)
      .pipe(finalize(() => this.savingName.set(false)))
      .subscribe({
        next: (user) => {
          this.name = user.name;
          this.profileSuccess.set('Nome atualizado.');
        },
        error: (err: HttpErrorResponse) =>
          this.profileError.set(apiErrorMessage(err, 'Falha ao salvar o nome.')),
      });
  }

  changePassword(): void {
    const invalid = this.passwordValidationError();
    if (invalid) {
      this.passwordError.set(invalid);
      return;
    }
    this.clearPasswordMessages();
    this.changingPassword.set(true);
    this.auth
      .changePassword({ currentPassword: this.currentPassword, newPassword: this.newPassword })
      .pipe(finalize(() => this.changingPassword.set(false)))
      .subscribe({
        next: () => {
          this.currentPassword = '';
          this.newPassword = '';
          this.confirmPassword = '';
          this.passwordSuccess.set('Senha alterada com sucesso.');
        },
        error: (err: HttpErrorResponse) => this.passwordError.set(passwordErrorMessage(err)),
      });
  }

  clearProfileMessages(): void {
    this.profileError.set('');
    this.profileSuccess.set('');
  }

  clearPasswordMessages(): void {
    this.passwordError.set('');
    this.passwordSuccess.set('');
  }

  private passwordValidationError(): string {
    if (this.currentPassword.length === 0) {
      return 'Informe a senha atual.';
    }
    if (!isValidPassword(this.newPassword)) {
      return `A nova senha deve ter de ${MIN_PASSWORD_LENGTH} a ${MAX_PASSWORD_LENGTH} caracteres.`;
    }
    if (this.newPassword !== this.confirmPassword) {
      return 'A confirmação não confere com a nova senha.';
    }
    if (this.newPassword === this.currentPassword) {
      return 'A nova senha deve ser diferente da atual.';
    }
    return '';
  }
}

function passwordErrorMessage(err: HttpErrorResponse): string {
  switch (err.status) {
    case 403:
      return 'Senha atual incorreta.';
    case 400:
      return 'A nova senha não atende aos requisitos.';
    default:
      return apiErrorMessage(err, 'Falha ao alterar a senha.');
  }
}
