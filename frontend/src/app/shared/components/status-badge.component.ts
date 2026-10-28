import { Component, input } from '@angular/core';

@Component({
  selector: 'app-status-badge',
  template: `<span class="badge" [class]="status()">{{ label() }}</span>`,
  styles: [
    `
      .badge {
        display: inline-block;
        padding: 0.15rem 0.55rem;
        border-radius: 999px;
        font-size: 0.72rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.06em;
        border: 1px solid var(--border);
      }
      .todo { color: var(--warn); }
      .doing { color: var(--accent); }
      .done { color: var(--ok); }
    `,
  ],
})
export class StatusBadgeComponent {
  readonly status = input.required<string>();

  label(): string {
    const labels: Record<string, string> = {
      todo: 'A fazer',
      doing: 'Em progresso',
      done: 'Concluída',
    };
    return labels[this.status()] ?? this.status();
  }
}
