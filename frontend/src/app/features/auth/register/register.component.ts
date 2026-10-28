import { Component, inject, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { AuthService } from '../../../core/services/auth.service';
import { isValidEmail, isValidPassword, MIN_PASSWORD_LENGTH } from '../auth-form.utils';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink],
  template: `
    <div class="auth-wrap">
      <form class="card stack" #form="ngForm" (ngSubmit)="submit(form)" novalidate>
        <div>
          <h1>Criar conta</h1>
          <p class="muted">Registre-se para começar a organizar suas tarefas.</p>
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
            aria-describedby="register-email-error"
          />
          @if ((emailModel.touched || submitted()) && emailModel.invalid) {
            <span id="register-email-error" class="error">Informe um e-mail válido.</span>
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
            [minlength]="minPasswordLength"
            maxlength="72"
            autocomplete="new-password"
            aria-describedby="register-password-error"
          />
          @if ((passwordModel.touched || submitted()) && passwordModel.invalid) {
            <span id="register-password-error" class="error">Use pelo menos 8 caracteres.</span>
          }
        </label>

        @if (error()) {
          <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
        }

        <button class="primary" type="submit" [disabled]="loading() || form.invalid">
          {{ loading() ? 'Criando...' : 'Criar conta' }}
        </button>

        <p class="muted">
          Já possui conta? <a routerLink="/login">Entrar</a>
        </p>
      </form>
    </div>
  `,
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
      next: () => this.router.navigate(['/tasks']),
      error: (err: HttpErrorResponse) => {
        this.error.set(
          err.status === 409 ? 'E-mail já cadastrado.' : 'Falha ao criar conta.',
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
