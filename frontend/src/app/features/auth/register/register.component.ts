import { Component, inject, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { AuthService } from '../../../core/services/auth.service';
import { isValidEmail, isValidPassword, MIN_PASSWORD_LENGTH } from '../auth-form.utils';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink],
  template: `
    <div class="auth">
      <aside class="panel">
        <span class="wordmark">atelier.</span>
        <div class="panel-copy">
          <h2>Uma ideia.<br />Um plano.<br />Um começo.</h2>
          <p>Seu trabalho merece um lugar sem distrações.</p>
        </div>
        <div class="panel-foot">
          <hr />
          <span>ORGANIZE. PRIORIZE. CONCLUA.</span>
        </div>
      </aside>

      <main class="auth-main">
        <form class="auth-form" #form="ngForm" (ngSubmit)="submit(form)" novalidate>
          <span class="eyebrow">CRIAR CONTA</span>
          <h1>Tudo começa<br />com clareza.</h1>

          <label class="field">
            <span class="field-label">E-mail</span>
            <input
              class="input"
              name="email"
              type="email"
              [(ngModel)]="email"
              (ngModelChange)="clearError()"
              #emailModel="ngModel"
              required
              email
              autocomplete="email"
              placeholder="seuemail@exemplo.com"
              aria-describedby="register-email-error"
            />
            @if ((emailModel.touched || submitted()) && emailModel.invalid) {
              <span id="register-email-error" class="error">Informe um e-mail válido.</span>
            }
          </label>

          <label class="field">
            <span class="field-label">Senha</span>
            <input
              class="input"
              name="password"
              type="password"
              [(ngModel)]="password"
              (ngModelChange)="clearError()"
              #passwordModel="ngModel"
              required
              [minlength]="minPasswordLength"
              maxlength="72"
              autocomplete="new-password"
              placeholder="Crie uma senha segura"
              aria-describedby="register-password-error"
            />
            @if ((passwordModel.touched || submitted()) && passwordModel.invalid) {
              <span id="register-password-error" class="error">Use pelo menos 8 caracteres.</span>
            }
          </label>

          @if (error()) {
            <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
          }

          <button class="btn btn-primary submit" type="submit" [disabled]="loading()">
            {{ loading() ? 'Criando...' : 'Criar minha conta' }}
            <span class="arrow" aria-hidden="true">↗</span>
          </button>

          <p class="switch">
            Já tem conta? <a routerLink="/login">ENTRAR →</a>
          </p>
        </form>
        <span class="meta">POST /register · 201 / 409</span>
      </main>
    </div>
  `,
  styles: [
    `
      .auth { min-height: 100vh; display: grid; grid-template-columns: minmax(0, 0.82fr) minmax(0, 1fr); background: var(--paper); }
      .panel {
        display: grid;
        grid-template-rows: auto 1fr auto;
        padding: 48px 56px;
        background: var(--pine);
        color: var(--paper-raised);
      }
      .wordmark { font-family: var(--font-display); font-weight: 700; font-size: 29px; line-height: 1; }
      .panel-copy { display: grid; gap: 26px; align-content: center; transform: translateY(-50px); }
      .panel-copy h2 {
        font-family: var(--font-display);
        font-weight: 700;
        font-size: clamp(2.8rem, 4.4vw, 64px);
        line-height: 1.06;
        letter-spacing: -0.03em;
        text-transform: uppercase;
        color: var(--lime);
      }
      .panel-copy p { font-size: 20px; line-height: 1.6; color: var(--paper-raised); }
      .panel-foot { display: grid; gap: 18px; }
      .panel-foot hr { border: 0; height: 1px; background: rgba(216, 221, 210, 0.45); margin: 0; }
      .panel-foot span { font-size: 11px; letter-spacing: 0.1em; color: var(--lime); }
      .auth-main {
        position: relative;
        display: flex;
        flex-direction: column;
        justify-content: flex-start;
        padding: 160px 56px 88px;
      }
      .auth-form { width: min(538px, 100%); display: grid; gap: 22px; align-content: start; }
      .auth-form h1 {
        font-size: clamp(2rem, 3vw, 43px);
        line-height: 1.08;
        letter-spacing: -0.03em;
        margin: 6px 0 8px;
      }
      .field { gap: 10px; }
      .submit { width: 100%; margin-top: 8px; }
      .switch { font-size: 13px; color: var(--muted); margin-top: 6px; }
      .switch a { color: var(--pine); font-size: 12px; font-weight: 500; letter-spacing: 0.04em; }
      .switch a:hover { text-decoration: underline; text-underline-offset: 3px; }
      .meta {
        position: absolute;
        left: 56px;
        bottom: 40px;
        font-size: 11px;
        color: var(--muted);
      }
      @media (max-width: 900px) {
        .auth { grid-template-columns: 1fr; }
        .panel { padding: 28px 22px; min-height: 0; grid-template-rows: auto auto auto; gap: 40px; }
        .panel-copy { gap: 18px; transform: none; }
        .panel-copy h2 { font-size: 40px; }
        .panel-copy p { font-size: 15px; }
        .panel-foot { display: none; }
        .auth-main { padding: 32px 22px 56px; }
        .auth-form h1 { font-size: 34px; }
        .meta { position: static; margin-top: 32px; }
      }
    `,
  ],
})
export class RegisterComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly minPasswordLength = MIN_PASSWORD_LENGTH;
  email = '';
  password = '';
  readonly loading = signal(false);
  readonly error = signal('');
  readonly submitted = signal(false);

  submit(form?: NgForm): void {
    this.submitted.set(true);
    if (form?.invalid || !this.validCredentials()) {
      form?.form.markAllAsTouched();
      this.error.set('Revise os campos destacados.');
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.auth.register({ email: this.email, password: this.password }).subscribe({
      next: () => void this.router.navigate(['/dashboard']),
      error: (err: HttpErrorResponse) => {
        this.error.set(
          err.status === 409
            ? 'E-mail já cadastrado.'
            : apiErrorMessage(err, 'Falha ao criar conta.'),
        );
        this.loading.set(false);
      },
    });
  }

  clearError(): void {
    this.error.set('');
  }

  private validCredentials(): boolean {
    return isValidEmail(this.email) && isValidPassword(this.password);
  }
}
