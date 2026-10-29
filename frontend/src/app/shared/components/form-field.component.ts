import { Component, input } from '@angular/core';

@Component({
  selector: 'app-form-field',
  template: `
    <div class="field">
      <label class="field-label" [for]="inputId()">
        <span class="desktop-field-label">{{ label() }}</span>
        <span class="mobile-field-label">{{ mobileLabel() || label() }}</span>
      </label>
      <ng-content />
      <ng-content select="[fieldError]" />
    </div>
  `,
  styles: [`
    :host { display: block; }
    .field { gap: var(--field-gap, 10px); }
    .mobile-field-label { display: none; }
    @media (max-width: 900px) {
      .desktop-field-label { display: none; }
      .mobile-field-label { display: inline; }
    }
  `],
})
export class FormFieldComponent {
  readonly label = input.required<string>();
  readonly inputId = input.required<string>();
  readonly mobileLabel = input('');
}
