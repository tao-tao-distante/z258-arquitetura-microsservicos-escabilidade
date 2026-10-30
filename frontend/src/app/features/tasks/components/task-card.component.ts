import { Component, computed, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';
import { STATUS_META, STATUSES, Task, TaskFeedback, TaskStatus } from '../models/task.model';
import { DueDateComponent } from './due-date.component';
import { PriorityBadgeComponent } from './priority-badge.component';

@Component({
  selector: 'app-task-card',
  imports: [RouterLink, PriorityBadgeComponent, DueDateComponent],
  template: `
    <article class="task" [class.saved]="feedback() === 'saved'" [class.failed]="feedback() === 'failed'">
      <h3>{{ task().title }}</h3>
      <p class="meta">Tarefa #{{ task().id.toString().padStart(3, '0') }}</p>
      <div class="tags">
        <app-priority-badge [priority]="task().priority" />
        <app-due-date [task]="task()" />
      </div>
      <hr />
      <div class="actions">
        <a class="link-arrow" [routerLink]="['/tasks', task().id]">
          ABRIR DETALHES <span aria-hidden="true">↗</span>
        </a>
        <label class="move">
          <span class="sr-only">Mover {{ task().title }} para</span>
          <select
            (change)="choose($event)"
            (mousedown)="$event.stopPropagation()"
            (touchstart)="$event.stopPropagation()"
          >
            <option value="" disabled selected>MOVER</option>
            @for (status of targets(); track status) {
              <option [value]="status">{{ label(status) }}</option>
            }
          </select>
        </label>
      </div>
    </article>
  `,
  styles: [
    `
      :host { display: block; cursor: grab; }
      .task {
        display: grid;
        gap: 10px;
        min-height: 138px;
        padding: 16px 14px 13px;
        background: var(--paper);
        border: 1px solid var(--line);
        border-radius: 3px;
        transition: border-color 0.15s ease;
      }
      .task:hover { border-color: var(--pine); }
      .task.saved { animation: saved-pulse 450ms ease-out; }
      .task.failed { animation: failed-shake 420ms cubic-bezier(0.36, 0.07, 0.19, 0.97); border-color: var(--danger); }
      h3 { font-size: 17px; line-height: 1.25; }
      .meta { font-size: 12px; color: var(--muted); }
      .tags { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; }
      hr { border: 0; height: 1px; background: var(--line); margin: 2px 0; }
      .actions { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
      .link-arrow { font-size: 10px; letter-spacing: 0.06em; }
      .move select {
        appearance: none;
        padding: 3px 8px;
        font: inherit;
        font-size: 10px;
        letter-spacing: 0.06em;
        color: var(--ink);
        background: transparent;
        border: 1px solid var(--line);
        border-radius: 3px;
        cursor: pointer;
      }
      .move select:hover,
      .move select:focus-visible { border-color: var(--pine); }
      @keyframes saved-pulse {
        0% { box-shadow: 0 0 0 0 var(--lime); border-color: var(--pine); }
        100% { box-shadow: 0 0 0 10px transparent; }
      }
      @keyframes failed-shake {
        10%, 90% { translate: -1px 0; }
        20%, 80% { translate: 2px 0; }
        30%, 50%, 70% { translate: -4px 0; }
        40%, 60% { translate: 4px 0; }
      }
    `,
  ],
})
export class TaskCardComponent {
  readonly task = input.required<Task>();
  readonly feedback = input<TaskFeedback | null>(null);
  readonly move = output<TaskStatus>();

  readonly targets = computed(() => STATUSES.filter((status) => status !== this.task().status));

  label(status: TaskStatus): string {
    return STATUS_META[status].label;
  }

  choose(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const status = select.value as TaskStatus;
    select.value = '';
    if (status) {
      this.move.emit(status);
    }
  }
}
