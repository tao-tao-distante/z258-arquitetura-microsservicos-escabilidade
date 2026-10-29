import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { TaskCardComponent } from '../components/task-card.component';
import { STATUS_META, STATUSES, Task, TaskStatus } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';

@Component({
  selector: 'app-task-board-page',
  imports: [RouterLink, TaskCardComponent],
  template: `
    <section class="page">
      <header class="head">
        <div>
          <h1 class="page-title">Do plano à entrega.</h1>
          <p class="page-subtitle">Arraste mentalmente o trabalho de uma etapa para outra.</p>
        </div>
        <a class="btn btn-lime" routerLink="/tasks/new">
          Nova tarefa <span class="arrow" aria-hidden="true">↗</span>
        </a>
      </header>

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }

      <div class="board">
        @for (status of statuses; track status) {
          <section class="column" [attr.aria-label]="meta(status).label">
            <div class="column-head" [class]="'column-head ' + status">
              <span class="eyebrow">{{ meta(status).index }} / {{ meta(status).short }}</span>
              <span class="count">{{ byStatus(status).length.toString().padStart(2, '0') }}</span>
            </div>
            <div class="column-body">
              @for (task of byStatus(status); track task.id) {
                <app-task-card [task]="task" />
              }
              @if (byStatus(status).length === 0) {
                <p class="empty">Nenhuma tarefa neste estágio.</p>
              }
            </div>
          </section>
        }
      </div>

      <p class="note">
        A movimentação no Angular será via <code>PUT /tasks/&#123;id&#125;</code> (status todo / doing / done).
      </p>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 26px; }
      .head { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; }
      .board { display: grid; grid-template-columns: repeat(3, minmax(240px, 1fr)); gap: 32px; align-items: stretch; }
      .column { display: grid; grid-template-rows: auto 1fr; }
      .column-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        padding: 12px 16px;
        border-radius: 3px 3px 0 0;
      }
      .column-head.todo { background: var(--band-todo); }
      .column-head.doing { background: var(--band-doing); }
      .column-head.done { background: var(--band-done); }
      .column-head .eyebrow { font-size: 10px; color: var(--ink); }
      .count { font-size: 11px; color: var(--ink); font-variant-numeric: tabular-nums; }
      .column-body {
        display: grid;
        gap: 10px;
        padding: 14px;
        background: var(--paper-raised);
        border: 1px solid var(--line);
        border-top: 0;
        border-radius: 0 0 3px 3px;
        min-height: 380px;
        align-content: start;
      }
      .empty { padding: 16px 4px; font-size: 12px; color: var(--muted); }
      .note { font-size: 11.5px; color: var(--muted); }
      .note code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
      @media (max-width: 900px) {
        .head { flex-direction: column; }
        .board {
          display: flex;
          overflow-x: auto;
          gap: 12px;
          padding-bottom: 8px;
          scroll-snap-type: x mandatory;
        }
        .column { flex: 0 0 82%; scroll-snap-align: start; }
        .column-body { min-height: 300px; }
      }
    `,
  ],
})
export class TaskBoardPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);

  readonly statuses = STATUSES;
  readonly tasks = signal<Task[]>([]);
  readonly error = signal('');

  private readonly grouped = computed(() => {
    const map: Record<TaskStatus, Task[]> = { todo: [], doing: [], done: [] };
    for (const task of this.tasks()) {
      map[task.status].push(task);
    }
    return map;
  });

  ngOnInit(): void {
    this.data.list().subscribe({
      next: (tasks) => this.tasks.set(tasks),
      error: (err: HttpErrorResponse) =>
        this.error.set(apiErrorMessage(err, 'Não foi possível carregar o quadro.')),
    });
  }

  byStatus(status: TaskStatus): Task[] {
    return this.grouped()[status];
  }

  meta(status: TaskStatus) {
    return STATUS_META[status];
  }
}
