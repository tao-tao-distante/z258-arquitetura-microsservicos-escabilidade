import { Component, input } from '@angular/core';

@Component({
  selector: 'app-auth-layout',
  template: `
    <div class="auth">
      <aside class="panel" [class.register]="variant() === 'register'">
        <span class="wordmark">atelier.</span>
        <div class="panel-copy">
          <h2>
            @for (line of panelLines(); track line) { <span>{{ line }}</span> }
          </h2>
          <p>{{ panelDescription() }}</p>
        </div>
        @if (showPanelFooter()) {
          <div class="panel-foot"><hr /><span>ORGANIZE. PRIORIZE. CONCLUA.</span></div>
        }
      </aside>
      <main class="auth-main" [class.register]="variant() === 'register'">
        <ng-content />
        <span class="meta">{{ meta() }}</span>
      </main>
    </div>
  `,
  styles: [`
    :host { display: block; }
    .auth { min-height: 100vh; display: grid; grid-template-columns: 650px minmax(0, 1fr); background: var(--paper); }
    .panel { position: relative; display: grid; grid-template-rows: auto 1fr auto; padding: 42px 57px 45px; background: var(--pine); color: var(--paper-raised); }
    .wordmark { font-family: var(--font-display); font-weight: 700; font-size: 31px; line-height: 1; }
    .panel-copy { position: absolute; top: 266px; left: 57px; right: 24px; }
    .panel-copy h2 { display: grid; margin: 0; font-size: 62px; line-height: 1.02; letter-spacing: -0.025em; text-transform: uppercase; color: var(--lime); }
    .panel-copy p { margin-top: 103px; max-width: 480px; font-size: 19px; line-height: 1.35; color: #d4e0d3; }
    .panel.register .panel-copy { top: 212px; }
    .panel.register .panel-copy h2 { font-size: 54px; }
    .panel.register .panel-copy p { margin-top: 186px; font-size: 18px; }
    .panel-foot { display: grid; gap: 20px; }
    .panel-foot hr { width: 100%; border: 0; height: 1px; background: rgba(216, 221, 210, 0.65); margin: 0; }
    .panel-foot span { font-size: 12px; color: var(--lime); }
    .auth-main { position: relative; padding: 166px 87px 88px; }
    .auth-main.register { padding-top: 142px; }
    .meta { position: absolute; left: 87px; bottom: 40px; font-size: 11px; color: var(--muted); }
    @media (max-width: 900px) {
      .auth { grid-template-columns: 1fr; }
      .panel { padding: 28px 22px; min-height: 250px; grid-template-rows: auto 1fr; }
      .panel-copy { position: static; align-self: end; }
      .panel-copy h2 { font-size: 38px; }
      .panel-copy p, .panel-foot { display: none; }
      .panel.register .panel-copy h2 { font-size: 38px; }
      .auth-main, .auth-main.register { padding: 36px 22px 72px; }
      .meta { position: static; display: block; margin-top: 36px; }
    }
  `],
})
export class AuthLayoutComponent {
  readonly panelLines = input.required<string[]>();
  readonly panelDescription = input.required<string>();
  readonly meta = input.required<string>();
  readonly variant = input<'login' | 'register'>('login');
  readonly showPanelFooter = input(true);
}
