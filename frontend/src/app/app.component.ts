import { Component, computed, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService } from './core/services/auth.service';

interface NavItem {
  index: string;
  label: string;
  path: string;
  exact?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { index: '01', label: 'Visão geral', path: '/dashboard', exact: true },
  { index: '02', label: 'Todas as tarefas', path: '/tasks', exact: true },
  { index: '03', label: 'Quadro', path: '/tasks/board', exact: true },
  { index: '04', label: 'Atividades', path: '/activities', exact: true },
];

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    @if (isAuthPage()) {
      <router-outlet />
    } @else {
      <div class="shell">
        <aside class="rail" [class.open]="menuOpen()" aria-label="Navegação principal">
          <div class="rail-top">
            <a routerLink="/dashboard" class="wordmark" (click)="closeMenu()">atelier.</a>
            <p class="workspace">/ WORKSPACE PESSOAL</p>
            <hr class="rail-rule" />
            <nav class="rail-nav">
              @for (item of navItems; track item.path) {
                <a
                  [routerLink]="item.path"
                  routerLinkActive="active"
                  [routerLinkActiveOptions]="{ exact: item.exact === true }"
                  (click)="closeMenu()"
                >
                  <span class="index">{{ item.index }}</span>
                  <span>{{ item.label }}</span>
                </a>
              }
            </nav>
          </div>

          <div class="rail-bottom">
            <p class="project">PROJETO / 2026</p>
            <hr class="rail-rule" />
            <div class="account">
              <span class="initials">{{ initials() }}</span>
              <span class="account-label">MINHA CONTA</span>
            </div>
            <button type="button" class="logout" (click)="logout()">Sair →</button>
          </div>
        </aside>

        @if (menuOpen()) {
          <button
            type="button"
            class="scrim"
            aria-label="Fechar menu"
            (click)="closeMenu()"
          ></button>
        }

        <div class="main">
          <div class="mobile-topbar">
            <a routerLink="/dashboard" class="wordmark">atelier.</a>
            <button
              type="button"
              class="menu-toggle"
              [attr.aria-expanded]="menuOpen()"
              aria-label="Abrir menu"
              (click)="toggleMenu()"
            >
              ☰
            </button>
          </div>

          <div class="page-top">
            <span class="crumb">{{ crumb() }}</span>
            <span class="today">{{ today }}</span>
          </div>
          <hr class="rule shell-rule" />
          <main class="page-content">
            <router-outlet />
          </main>
        </div>

        <nav class="mobile-nav" aria-label="Navegação inferior">
          @for (item of mobileItems; track item.path) {
            <a
              [routerLink]="item.path"
              routerLinkActive="active"
              [routerLinkActiveOptions]="{ exact: true }"
            >{{ item.label }}</a>
          }
        </nav>
      </div>
    }
  `,
  styles: [
    `
      .shell {
        display: flex;
        min-height: 100vh;
        background: var(--paper);
      }

      .rail {
        width: 240px;
        flex: 0 0 240px;
        background: var(--pine);
        color: var(--paper-raised);
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        padding: 29px 20px 34px;
        position: sticky;
        top: 0;
        height: 100vh;
      }

      .wordmark {
        display: inline-block;
        font-family: var(--font-display);
        font-weight: 700;
        font-size: 29px;
        line-height: 1;
        color: var(--paper-raised);
        padding-left: 11px;
      }

      .workspace {
        margin: 33px 0 17px;
        padding-left: 11px;
        font-size: 10px;
        font-weight: 500;
        letter-spacing: 0.09em;
        color: var(--lime);
      }

      .rail-rule {
        border: 0;
        height: 1px;
        background: rgba(216, 221, 210, 0.45);
        margin: 0 8px 25px 8px;
      }

      .rail-nav {
        display: grid;
        gap: 15px;
      }

      .rail-nav a {
        display: flex;
        align-items: center;
        gap: 14px;
        min-height: 43px;
        padding: 0 10px;
        border-radius: 3px;
        font-size: 13px;
        font-weight: 500;
        color: var(--rail-text);
      }

      .rail-nav a:hover {
        color: var(--paper-raised);
      }

      .rail-nav a.active {
        background: var(--pine-soft);
        color: var(--paper-raised);
      }

      .index {
        font-variant-numeric: tabular-nums;
      }

      .rail-bottom {
        display: grid;
        gap: 14px;
      }

      .project {
        padding-left: 11px;
        font-size: 11px;
        letter-spacing: 0.08em;
        color: var(--lime);
      }

      .account {
        display: flex;
        align-items: center;
        gap: 14px;
        padding-left: 11px;
      }

      .initials {
        font-size: 16px;
        font-weight: 500;
        color: var(--lime);
      }

      .account-label {
        font-size: 11px;
        font-weight: 500;
        color: var(--paper-raised);
      }

      .logout {
        justify-self: start;
        padding: 0 0 0 11px;
        border: 0;
        background: none;
        color: var(--rail-muted);
        font-size: 12px;
      }

      .logout:hover {
        color: var(--paper-raised);
      }

      .main {
        flex: 1 1 auto;
        min-width: 0;
        display: flex;
        flex-direction: column;
      }

      .page-top {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        padding: 0 42px;
        min-height: 76px;
      }

      .crumb {
        font-size: 11px;
        font-weight: 500;
        letter-spacing: 0.09em;
        color: var(--muted);
      }

      .today {
        font-size: 12px;
        font-weight: 500;
        color: var(--ink);
        white-space: nowrap;
      }

      .page-content {
        padding: 30px 42px 56px;
        max-width: 1148px;
        width: 100%;
      }

      .shell-rule { margin: 0 49px 0 41px; }

      .mobile-topbar,
      .mobile-nav,
      .scrim {
        display: none;
      }

      @media (max-width: 900px) {
        .shell {
          display: block;
          padding-bottom: 54px;
        }

        .rail {
          position: fixed;
          inset: 0 auto 0 0;
          width: 264px;
          height: 100dvh;
          z-index: 40;
          transform: translateX(-100%);
          transition: transform 0.2s ease-out;
        }

        .rail.open {
          transform: translateX(0);
        }

        .scrim {
          display: block;
          position: fixed;
          inset: 0;
          z-index: 30;
          border: 0;
          background: rgba(24, 61, 53, 0.45);
        }

        .mobile-topbar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          height: 68px;
          padding: 0 22px;
          background: var(--pine);
        }

        .mobile-topbar .wordmark {
          font-size: 23px;
          padding-left: 0;
        }

        .menu-toggle {
          border: 0;
          background: none;
          color: var(--paper-raised);
          font-size: 22px;
          line-height: 1;
          padding: 8px;
          min-width: 44px;
          min-height: 44px;
        }

        .page-top, .shell-rule { display: none; }

        .page-content {
          padding: 35px 22px 96px;
          max-width: none;
        }

        .mobile-nav {
          display: flex;
          position: fixed;
          inset: auto 0 0 0;
          height: 54px;
          align-items: center;
          justify-content: space-around;
          gap: 1rem;
          padding: 0 22px;
          background: var(--paper-raised);
          z-index: 20;
        }

        .mobile-nav a {
          font-size: 11px;
          font-weight: 500;
          letter-spacing: 0.08em;
          color: var(--muted);
        }

        .mobile-nav a.active {
          color: var(--pine);
        }
      }
    `,
  ],
})
export class AppComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly navItems = NAV_ITEMS;
  readonly mobileItems: NavItem[] = [
    { index: '01', label: 'INÍCIO', path: '/dashboard' },
    { index: '02', label: 'TAREFAS', path: '/tasks' },
    { index: '03', label: 'QUADRO', path: '/tasks/board' },
  ];

  readonly menuOpen = signal(false);
  readonly currentUrl = signal(this.router.url);
  readonly isAuthPage = computed(() => {
    const url = this.currentUrl();
    return url.startsWith('/login') || url.startsWith('/register');
  });
  readonly crumb = computed(() => breadcrumbFor(this.currentUrl()));
  readonly initials = computed(() => {
    const email = this.auth.user()?.email ?? '';
    return email.slice(0, 2).toUpperCase() || 'VC';
  });
  readonly today = formatToday();

  constructor() {
    this.router.events
      .pipe(filter((event): event is NavigationEnd => event instanceof NavigationEnd))
      .subscribe((event) => {
        this.currentUrl.set(event.urlAfterRedirects);
        this.closeMenu();
      });
  }

  toggleMenu(): void {
    this.menuOpen.update((open) => !open);
  }

  closeMenu(): void {
    this.menuOpen.set(false);
  }

  logout(): void {
    this.closeMenu();
    this.auth.logout();
  }
}

function breadcrumbFor(url: string): string {
  const path = url.split('?')[0] ?? url;
  if (path === '/dashboard') {
    return 'WORKSPACE / DASHBOARD / VISÃO GERAL';
  }
  if (path === '/tasks') {
    return 'WORKSPACE / TAREFAS / LISTA';
  }
  if (path === '/tasks/board') {
    return 'WORKSPACE / KANBAN / QUADRO';
  }
  if (path === '/tasks/new') {
    return 'WORKSPACE / CRIAR TAREFA / FORMULÁRIO';
  }
  if (/^\/tasks\/[^/]+\/edit$/.test(path)) {
    return 'WORKSPACE / EDITAR TAREFA / FORMULÁRIO';
  }
  if (/^\/tasks\/[^/]+$/.test(path)) {
    return 'WORKSPACE / DETALHE DA TAREFA';
  }
  if (path === '/activities') {
    return 'WORKSPACE / ATIVIDADES';
  }
  return 'WORKSPACE';
}

function formatToday(): string {
  const months = ['JAN', 'FEV', 'MAR', 'ABR', 'MAI', 'JUN', 'JUL', 'AGO', 'SET', 'OUT', 'NOV', 'DEZ'];
  const now = new Date();
  const day = now.getDate().toString().padStart(2, '0');
  return `${day} ${months[now.getMonth()]} ${now.getFullYear()}`;
}
