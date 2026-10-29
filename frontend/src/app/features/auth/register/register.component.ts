import { Component, inject, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { AuthService } from '../../../core/services/auth.service';
import { AuthLayoutComponent } from '../../../shared/components/auth-layout.component';
import { FormFieldComponent } from '../../../shared/components/form-field.component';
import { isValidEmail, isValidPassword, MIN_PASSWORD_LENGTH } from '../auth-form.utils';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink, AuthLayoutComponent, FormFieldComponent],
  template: `
    <app-auth-layout
      [panelLines]="['UMA IDEIA.', 'UM PLANO.', 'UM COMEÇO.']"
      panelDescription="Seu trabalho merece um lugar sem distrações."
      meta="POST /register · 201 / 409"
      variant="register"
      [showPanelFooter]="false"
    >
      <form class="auth-form" #form="ngForm" (ngSubmit)="submit(form)" novalidate animate.enter="page-enter">
          <span class="eyebrow">CRIAR CONTA</span>
          <h1>Tudo começa<br />com clareza.</h1>

          <app-form-field label="E-MAIL" inputId="register-email">
            <input
              id="register-email"
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
              <span fieldError id="register-email-error" class="error">Informe um e-mail válido.</span>
            }
          </app-form-field>

          <app-form-field label="SENHA" inputId="register-password">
            <input
              id="register-password"
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
              <span fieldError id="register-password-error" class="error">Use pelo menos 8 caracteres.</span>
            }
          </app-form-field>

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
    </app-auth-layout>
  `,
  styles: [
    `
      .auth-form { width: min(558px, 100%); display: grid; gap: 0; align-content: start; }
      .auth-form h1 {
        font-size: clamp(2rem, 3vw, 43px);
        line-height: 1.08;
        letter-spacing: -0.03em;
        margin: 16px 0 39px;
      }
      app-form-field + app-form-field { margin-top: 35px; }
      .submit { width: 100%; margin-top: 49px; }
      .switch { font-size: 13px; color: var(--muted); margin-top: 31px; }
      .switch a { color: var(--pine); font-size: 12px; font-weight: 500; letter-spacing: 0.04em; }
      .switch a:hover { text-decoration: underline; text-underline-offset: 3px; }
      @media (max-width: 900px) {
        .auth-form h1 { font-size: 34px; }
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
