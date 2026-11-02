import { inject, Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { tap } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  AuthResponse,
  AuthUser,
  Credentials,
  PasswordChange,
  ProfileUpdate,
} from '../models/auth.model';

const TOKEN_KEY = 'tm_token';
const USER_KEY = 'tm_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly api = environment.identityUrl;

  readonly token = signal<string | null>(null);
  readonly user = signal<AuthUser | null>(null);
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

  loadProfile() {
    return this.http.get<AuthUser>(`${this.api}/me`).pipe(tap((user) => this.storeUser(user)));
  }

  updateName(name: string) {
    const body: ProfileUpdate = { name };
    return this.http
      .patch<AuthUser>(`${this.api}/me`, body)
      .pipe(tap((user) => this.storeUser(user)));
  }

  changePassword(change: PasswordChange) {
    return this.http.put<void>(`${this.api}/me/password`, {
      current_password: change.currentPassword,
      new_password: change.newPassword,
    });
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
    this.token.set(res.token);
    this.storeUser({ id: res.id, email: res.email, name: res.name });
  }

  private storeUser(user: AuthUser): void {
    const stored: AuthUser = { id: user.id, email: user.email, name: user.name };
    window.localStorage.setItem(USER_KEY, JSON.stringify(stored));
    this.user.set(stored);
  }

  private readUser(): AuthUser | null {
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
        const { id, email, name } = user as { id: number; email: string; name?: unknown };
        return { id, email, name: typeof name === 'string' ? name : '' };
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
