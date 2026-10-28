import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { NgForm } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthService } from '../../../core/services/auth.service';
import { isValidEmail, isValidPassword, safeReturnUrl } from '../auth-form.utils';

@Component({
  selector: 'app-login',
  imports: [FormsModule, RouterLink],
  template: `
    <div class="auth-wrap">
      <form class="card stack" #form="ngForm" (ngSubmit)="submit(form)" novalidate>
        <div>
          <h1>Entrar</h1>
          <p class="muted">Acesse o Task Manager para gerenciar suas tarefas.</p>
        </div>

        <label class="stack field">
          <span class="muted">E-mail</span>
          <input
            name="email"
            type="email"
            [(ngModel)]="email"
            (ngModelChange)="clearError()"
            #emailModel="ngModel"
            required
            email
            autocomplete="email"
            aria-describedby="login-email-error"
          />
          @if ((emailModel.touched || submitted()) && emailModel.invalid) {
            <span id="login-email-error" class="error">Informe um e-mail válido.</span>
          }
        </label>

        <label class="stack field">
          <span class="muted">Senha</span>
          <input
            name="password"
            type="password"
            [(ngModel)]="password"
            (ngModelChange)="clearError()"
            #passwordModel="ngModel"
            required
            maxlength="72"
            autocomplete="current-password"
            aria-describedby="login-password-error"
          />
          @if ((passwordModel.touched || submitted()) && passwordModel.invalid) {
            <span id="login-password-error" class="error">Informe sua senha.</span>
          }
        </label>

        @if (error()) {
          <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
        }

        <button class="primary" type="submit" [disabled]="loading() || form.invalid">
          {{ loading() ? 'Entrando...' : 'Entrar' }}
        </button>

        <p class="muted">
          Não tem conta? <a routerLink="/register">Cadastre-se</a>
        </p>
      </form>
    </div>
  `,
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
        void this.router.navigateByUrl(safeReturnUrl(returnUrl));
      },
      error: (err: HttpErrorResponse) => {
        this.error.set(err.status === 401 ? 'Credenciais inválidas.' : 'Falha ao entrar.');
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
