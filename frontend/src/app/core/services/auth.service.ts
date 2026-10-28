import { computed, inject, Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { tap } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AuthResponse, Credentials } from '../models/auth.model';

const TOKEN_KEY = 'tm_token';
const USER_KEY = 'tm_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly api = environment.identityUrl;

  readonly token = signal<string | null>(null);
  readonly user = signal<{ id: number; email: string } | null>(null);
  readonly isAuthenticated = computed(() => this.token() !== null);

  constructor() {
    const token = window.localStorage.getItem(TOKEN_KEY);
    const user = this.readUser();
    if (token && user && this.isValidToken(token)) {
      this.token.set(token);
      this.user.set(user);
    } else if (token || user) {
      this.clearSession();
    }
  }

  login(credentials: Credentials) {
    return this.authenticate('/login', credentials);
  }

  register(credentials: Credentials) {
    return this.authenticate('/register', credentials);
  }

  logout(): void {
    this.clearSession();
    void this.router.navigate(['/login']);
  }

  clearSession(): void {
    window.localStorage.removeItem(TOKEN_KEY);
    window.localStorage.removeItem(USER_KEY);
    this.token.set(null);
    this.user.set(null);
  }

  hasValidSession(): boolean {
    return this.sessionToken() !== null && this.user() !== null;
  }

  sessionToken(): string | null {
    const token = this.token();
    if (token && this.isValidToken(token)) {
      return token;
    }
    if (token || this.user()) {
      this.clearSession();
    }
    return null;
  }

  private authenticate(path: string, credentials: Credentials) {
    return this.http
      .post<AuthResponse>(`${this.api}${path}`, credentials)
      .pipe(tap((res) => this.persist(res)));
  }

  private persist(res: AuthResponse): void {
    window.localStorage.setItem(TOKEN_KEY, res.token);
    window.localStorage.setItem(USER_KEY, JSON.stringify({ id: res.id, email: res.email }));
    this.token.set(res.token);
    this.user.set({ id: res.id, email: res.email });
  }

  private readUser(): { id: number; email: string } | null {
    const raw = window.localStorage.getItem(USER_KEY);
    if (!raw) {
      return null;
    }
    try {
      const user: unknown = JSON.parse(raw);
      if (
        typeof user === 'object' &&
        user !== null &&
        typeof (user as { id?: unknown }).id === 'number' &&
        typeof (user as { email?: unknown }).email === 'string'
      ) {
        return user as { id: number; email: string };
      }
      return null;
    } catch {
      return null;
    }
  }

  private isValidToken(token: string): boolean {
    const parts = token.split('.');
    if (parts.length !== 3 || parts.some((part) => !part)) {
      return false;
    }
    try {
      const base64 = parts[1].replaceAll('-', '+').replaceAll('_', '/');
      const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
      const payload: unknown = JSON.parse(atob(padded));
      const exp = (payload as { exp?: unknown } | null)?.exp;
      return typeof exp === 'number' && Number.isFinite(exp) && exp > Date.now() / 1000;
    } catch {
      return false;
    }
  }
}
