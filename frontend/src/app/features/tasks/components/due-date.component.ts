import { Component, computed, input } from '@angular/core';
import { Task } from '../models/task.model';
import { dueInfo } from '../task.utils';

@Component({
  selector: 'app-due-date',
  template: `
    @if (info(); as due) {
      <span [class]="'due ' + due.tone">
        @if (due.tone === 'overdue') { <span class="sr-only">Atenção: </span> }
        {{ due.label }}
      </span>
    }
  `,
  styles: [
    `
      :host { display: inline-flex; }
      .due { font-size: 11px; letter-spacing: 0.02em; color: var(--muted); white-space: nowrap; }
      .due.soon { color: var(--pine); font-weight: 500; }
      .due.overdue { color: var(--danger); font-weight: 600; }
    `,
  ],
})
export class DueDateComponent {
  readonly task = input.required<Pick<Task, 'due_date' | 'status'>>();
  /** Data de referência (YYYY-MM-DD); padrão é hoje no navegador. */
  readonly today = input<string | undefined>(undefined);

  readonly info = computed(() => dueInfo(this.task(), this.today()));
}
