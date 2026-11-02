import { Component, input } from '@angular/core';

@Component({
  selector: 'app-page-header',
  template: `
    <header class="page-header">
      <div class="desktop-copy">
        <h1 class="page-title">{{ title() }}</h1>
        <p class="page-subtitle">{{ subtitle() }}</p>
      </div>
      <div class="mobile-copy">
        <span class="eyebrow">{{ mobileEyebrow() }}</span>
        <h1>{{ mobileTitle() || title() }}</h1>
      </div>
      <div class="header-action" [class.mobile-hidden]="!showMobileAction()"><ng-content /></div>
    </header>
  `,
  styles: [`
    :host { display: block; }
    .page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; }
    .mobile-copy { display: none; }
    @media (max-width: 900px) {
      .page-header { display: grid; gap: 0; }
      .desktop-copy { display: none; }
      .mobile-copy { display: grid; gap: 16px; }
      .mobile-copy h1 { font-size: 36px; line-height: 1.05; letter-spacing: -0.025em; }
      .header-action { margin-top: 22px; }
      .header-action:empty { display: none; }
      .header-action.mobile-hidden { display: none; }
    }
  `],
})
export class PageHeaderComponent {
  readonly title = input.required<string>();
  readonly subtitle = input.required<string>();
  readonly mobileEyebrow = input('');
  readonly mobileTitle = input('');
  readonly showMobileAction = input(false);
}
