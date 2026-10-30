import { Component, computed, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Task } from '../models/task.model';
import { formatDateTime } from '../task.utils';
import { DueDateComponent } from './due-date.component';
import { PriorityBadgeComponent } from './priority-badge.component';
import { StatusBadgeComponent } from './status-badge.component';

@Component({
  selector: 'app-task-row',
  imports: [RouterLink, StatusBadgeComponent, PriorityBadgeComponent, DueDateComponent],
  template: `
    <a class="task-row" [routerLink]="['/tasks', task().id]">
      <span class="index">{{ index().toString().padStart(2, '0') }}</span>
      <span class="copy">
        <span class="title">{{ task().title }}</span>
        @if (task().description) { <span class="desc">{{ task().description }}</span> }
      </span>
      <app-priority-badge [priority]="task().priority" />
      <span class="due">
        @if (task().due_date) { <app-due-date [task]="task()" /> } @else { <span class="none">—</span> }
      </span>
      <span class="created">{{ created() }}</span>
      <app-status-badge [status]="task().status" [withIndex]="true" />
      <span class="arrow" aria-hidden="true">↗</span>
    </a>
  `,
  styles: [`
    :host { display: block; }
    .task-row { display: grid; grid-template-columns: var(--task-columns, 44px 1fr 280px 24px); align-items: center; gap: 16px; padding: 23px 4px; }
    .task-row:hover .title { text-decoration: underline; text-underline-offset: 3px; }
    .index { font-size: 11px; color: var(--muted); font-variant-numeric: tabular-nums; }
    .copy { display: grid; gap: 3px; min-width: 0; }
    .title { font-family: var(--font-display); font-weight: 700; font-size: 18px; color: var(--pine); }
    .desc { font-size: 12px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .none, .created { font-size: 11px; color: var(--muted); font-variant-numeric: tabular-nums; }
    .arrow { font-size: 20px; color: var(--pine); justify-self: end; }
  `],
})
export class TaskRowComponent {
  readonly task = input.required<Task>();
  readonly index = input.required<number>();

  /** Data de criação no fuso local, sem horário. */
  readonly created = computed(() => formatDateTime(this.task().created_at).slice(0, 10));
}
