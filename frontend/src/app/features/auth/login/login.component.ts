import { Component, inject, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { AuthService } from '../../../core/services/auth.service';
import { AuthLayoutComponent } from '../../../shared/components/auth-layout.component';
import { FormFieldComponent } from '../../../shared/components/form-field.component';
import { isValidEmail, isValidPassword, safeReturnUrl } from '../auth-form.utils';

@Component({
  selector: 'app-login',
  imports: [FormsModule, RouterLink, AuthLayoutComponent, FormFieldComponent],
  template: `
    <app-auth-layout
      [panelLines]="['MENOS RUÍDO.', 'MAIS FEITO.']"
      panelDescription="Uma forma mais clara de organizar o trabalho que importa."
      meta="JWT · POST /login"
    >
      <form class="auth-form" #form="ngForm" (ngSubmit)="submit(form)" novalidate animate.enter="page-enter">
          <span class="eyebrow">BEM-VINDO DE VOLTA</span>
          <h1>O próximo passo<br />começa aqui.</h1>

          <app-form-field label="E-MAIL" inputId="login-email">
            <input
              id="login-email"
              class="input"
              name="email"
              type="email"
              [(ngModel)]="email"
              (ngModelChange)="clearError()"
              #emailModel="ngModel"
              required
              email
              autocomplete="email"
              placeholder="nome@exemplo.com"
              aria-describedby="login-email-error"
            />
            @if ((emailModel.touched || submitted()) && emailModel.invalid) {
              <span fieldError id="login-email-error" class="error">Informe um e-mail válido.</span>
            }
          </app-form-field>

          <app-form-field label="SENHA" inputId="login-password">
            <input
              id="login-password"
              class="input"
              name="password"
              type="password"
              [(ngModel)]="password"
              (ngModelChange)="clearError()"
              #passwordModel="ngModel"
              required
              maxlength="72"
              autocomplete="current-password"
              placeholder="••••••••"
              aria-describedby="login-password-error"
            />
            @if ((passwordModel.touched || submitted()) && passwordModel.invalid) {
              <span fieldError id="login-password-error" class="error">Informe sua senha.</span>
            }
          </app-form-field>

          @if (error()) {
            <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
          }

          <button class="btn btn-primary submit" type="submit" [disabled]="loading()">
            {{ loading() ? 'Entrando...' : 'Entrar no workspace' }}
            <span class="arrow" aria-hidden="true">↗</span>
          </button>

          <p class="switch">
            Ainda não tem conta? <a routerLink="/register">CADASTRE-SE →</a>
          </p>
      </form>
    </app-auth-layout>
  `,
  styles: [
    `
      .auth-form { width: min(559px, 100%); display: grid; gap: 0; align-content: start; }
      .auth-form h1 {
        font-size: clamp(2rem, 3vw, 43px);
        line-height: 1.08;
        letter-spacing: -0.03em;
        margin: 17px 0 40px;
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
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

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
    this.auth.login({ email: this.email, password: this.password }).subscribe({
      next: () => {
        const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl');
        void this.router.navigateByUrl(safeReturnUrl(returnUrl, '/dashboard'));
      },
      error: (err: HttpErrorResponse) => {
        this.error.set(
          err.status === 401 ? 'Credenciais inválidas.' : apiErrorMessage(err, 'Falha ao entrar.'),
        );
        this.loading.set(false);
      },
    });
  }

  clearError(): void {
    this.error.set('');
  }

  private validCredentials(): boolean {
    return isValidEmail(this.email) && isValidPassword(this.password, 1);
  }
}
