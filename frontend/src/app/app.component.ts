import { Component, computed, inject } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { AuthService } from './core/services/auth.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <header class="topbar">
      <div class="container header-content">
        <a routerLink="/tasks" class="brand">
          <span class="logo">TM</span>
          <span>Task Manager</span>
        </a>

        @if (authenticated()) {
          <nav class="row" aria-label="Navegação principal">
            <a routerLink="/tasks" routerLinkActive="active">Kanban</a>
            <a routerLink="/notifications" routerLinkActive="active">Notificações</a>
            <button class="danger" (click)="logout()">Sair</button>
          </nav>
        }
      </div>
    </header>
    <main class="container">
      <router-outlet />
    </main>
  `,
  styles: [
    `
      .topbar {
        border-bottom: 1px solid var(--border);
        background: rgba(15, 17, 23, 0.85);
        backdrop-filter: blur(12px);
        position: sticky;
        top: 0;
        z-index: 10;
      }
      .brand {
        display: flex;
        align-items: center;
        gap: 0.6rem;
        color: var(--text);
        font-weight: 800;
        letter-spacing: 0.02em;
      }
      .brand:hover { text-decoration: none; }
      .logo {
        display: grid;
        place-items: center;
        width: 34px;
        height: 34px;
        border-radius: 10px;
        background: linear-gradient(135deg, var(--accent), var(--accent-2));
        font-size: 0.8rem;
      }
      nav { gap: 1rem; }
      nav a { color: var(--muted); font-weight: 600; }
      nav a.active, nav a:hover { color: var(--text); text-decoration: none; }
      .header-content { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
      @media (max-width: 600px) {
        .header-content { align-items: flex-start; flex-direction: column; padding-block: 0.9rem; }
        nav { width: 100%; justify-content: space-between; gap: 0.5rem; }
        nav a, nav button { font-size: 0.85rem; }
      }
    `,
  ],
})
export class AppComponent {
  private readonly auth = inject(AuthService);
  readonly authenticated = computed(() => this.auth.isAuthenticated());

  logout(): void {
    this.auth.logout();
  }
}
