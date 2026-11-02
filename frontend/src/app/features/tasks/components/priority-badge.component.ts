import { Component, computed, input } from '@angular/core';
import { PRIORITY_META, TaskPriority } from '../models/task.model';

@Component({
  selector: 'app-priority-badge',
  template: `<span [class]="'priority ' + priority()" [attr.title]="'Prioridade ' + label()">{{ short() }}</span>`,
  styles: [
    `
      :host { display: inline-flex; }
      .priority {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 2px 8px;
        border: 1px solid var(--line);
        border-radius: 2px;
        font-size: 10px;
        font-weight: 500;
        letter-spacing: 0.06em;
        color: var(--muted);
        white-space: nowrap;
      }
      .priority::before { content: ''; width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
      .priority.medium { color: var(--ink); }
      .priority.high { color: var(--pine); border-color: var(--pine); }
      .priority.urgent { color: var(--danger); border-color: var(--danger); background: var(--danger-soft); }
    `,
  ],
})
export class PriorityBadgeComponent {
  readonly priority = input.required<TaskPriority>();

  readonly label = computed(() => PRIORITY_META[this.priority()].label);
  readonly short = computed(() => PRIORITY_META[this.priority()].short);
}
