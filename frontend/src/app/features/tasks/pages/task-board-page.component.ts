import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { finalize } from 'rxjs';
import { TaskCardComponent } from '../components/task-card.component';
import { TaskDataService } from '../services/task-data.service';
import { STATUSES, Task, TaskStatus } from '../models/task.model';

const COLUMN_TITLES: Record<TaskStatus, string> = {
  todo: 'A fazer',
  doing: 'Em progresso',
  done: 'Concluídas',
};

@Component({
  selector: 'app-task-board-page',
  imports: [FormsModule, TaskCardComponent],
  template: `
    <section class="stack">
      <div class="row" style="justify-content: space-between; align-items: flex-end;">
        <div>
          <h1 style="margin: 0;">Kanban</h1>
          <p class="muted" style="margin: 0.25rem 0 0;">
            Organize suas tarefas por estágio de execução.
          </p>
        </div>
        <span class="muted">{{ tasks().length }} tarefa(s)</span>
      </div>

      <form class="card task-form" #form="ngForm" (ngSubmit)="create(form)" novalidate>
        <label class="stack field title-field">
          <span>Título</span>
          <input
            name="title"
            [(ngModel)]="newTitle"
            (ngModelChange)="clearError()"
            #titleModel="ngModel"
            required
            maxlength="160"
            aria-describedby="task-title-error"
          />
          @if ((titleModel.touched || createSubmitted()) && titleModel.invalid) {
            <span id="task-title-error" class="error">Informe o título da tarefa.</span>
          }
        </label>
        <label class="stack field description-field">
          <span>Descrição <span class="muted">(opcional)</span></span>
          <input
            name="description"
            [(ngModel)]="newDescription"
            (ngModelChange)="clearError()"
            maxlength="500"
          />
        </label>
        <button class="primary add-button" type="submit" [disabled]="createPending() || form.invalid">
          {{ createPending() ? 'Adicionando...' : 'Adicionar' }}
        </button>
      </form>

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }
      <p class="sr-only" aria-live="polite">{{ announcement() }}</p>

      <div class="board">
        @for (status of statuses; track status) {
          <div class="column">
            <div class="row" style="justify-content: space-between;">
              <h2>{{ columnTitle(status) }}</h2>
              <span class="count">{{ byStatus(status).length }}</span>
            </div>
            @for (task of byStatus(status); track task.id) {
              <app-task-card
                [task]="task"
                [pending]="isPending(task.id)"
                (advance)="advance($event)"
                (remove)="remove($event)"
              />
            }
            @if (byStatus(status).length === 0) {
              <p class="muted empty">Nenhuma tarefa.</p>
            }
          </div>
        }
      </div>
    </section>
  `,
  styles: [
    `
      .board {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
        gap: 1rem;
        align-items: start;
      }
      .column {
        display: grid;
        gap: 0.75rem;
        background: rgba(23, 26, 35, 0.6);
        border: 1px solid var(--border);
        border-radius: 16px;
        padding: 1rem;
        min-height: 220px;
      }
      h2 { margin: 0; font-size: 0.95rem; text-transform: uppercase; letter-spacing: 0.08em; }
      .count {
        background: var(--surface-2);
        border-radius: 999px;
        padding: 0.1rem 0.5rem;
        font-size: 0.75rem;
      }
      .empty { margin: 0; font-size: 0.85rem; }
      .task-form { display: grid; grid-template-columns: 2fr 3fr auto; gap: 0.75rem; align-items: start; }
      .field { gap: 0.4rem; }
      .field > span:first-child { font-size: 0.85rem; color: var(--muted); }
      .add-button { margin-top: 1.55rem; }
      @media (max-width: 700px) {
        .task-form { grid-template-columns: 1fr; }
        .add-button { margin-top: 0; width: 100%; }
      }
    `,
  ],
})
export class TaskBoardPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);

  readonly statuses = STATUSES;
  readonly tasks = signal<Task[]>([]);
  readonly createPending = signal(false);
  readonly createSubmitted = signal(false);
  readonly pendingTaskIds = signal<ReadonlySet<number>>(new Set());
  readonly error = signal('');
  readonly announcement = signal('');

  newTitle = '';
  newDescription = '';

  private readonly grouped = computed(() => {
    const map: Record<TaskStatus, Task[]> = { todo: [], doing: [], done: [] };
    for (const task of this.tasks()) {
      map[task.status].push(task);
    }
    return map;
  });

  ngOnInit(): void {
    this.reload();
  }

  byStatus(status: TaskStatus): Task[] {
    return this.grouped()[status];
  }

  columnTitle(status: TaskStatus): string {
    return COLUMN_TITLES[status];
  }

  create(form?: NgForm): void {
    this.createSubmitted.set(true);
    const title = this.newTitle.trim();
    if (form?.invalid || !title || this.createPending()) {
      form?.form.markAllAsTouched();
      if (!title) {
        this.error.set('Informe o título da tarefa.');
      }
      return;
    }
    this.error.set('');
    this.createPending.set(true);
    this.data.create({ title, description: this.newDescription.trim() }).pipe(
      finalize(() => this.createPending.set(false)),
    ).subscribe({
      next: (task) => {
        this.tasks.update((items) => [task, ...items]);
        this.newTitle = '';
        this.newDescription = '';
        this.createSubmitted.set(false);
        this.announcement.set(`Tarefa ${task.title} adicionada.`);
      },
      error: (err: HttpErrorResponse) => this.fail(err),
    });
  }

  advance(task: Task): void {
    if (!this.beginTaskAction(task.id)) {
      return;
    }
    const next: TaskStatus = task.status === 'todo' ? 'doing' : 'done';
    this.data
      .update(task.id, {
        title: task.title,
        description: task.description,
        status: next,
      }).pipe(finalize(() => this.endTaskAction(task.id)))
      .subscribe({
        next: (updated) => {
          this.tasks.update((items) =>
            items.map((item) => (item.id === updated.id ? updated : item)),
          );
          this.announcement.set(`Tarefa ${updated.title} atualizada.`);
        },
        error: (err: HttpErrorResponse) => this.fail(err),
      });
  }

  remove(task: Task): void {
    if (!this.beginTaskAction(task.id)) {
      return;
    }
    this.data.remove(task.id).pipe(finalize(() => this.endTaskAction(task.id))).subscribe({
      next: () => {
        this.tasks.update((items) => items.filter((item) => item.id !== task.id));
        this.announcement.set(`Tarefa ${task.title} excluída.`);
      },
      error: (err: HttpErrorResponse) => this.fail(err),
    });
  }

  private reload(): void {
    this.error.set('');
    this.data.list().subscribe({
      next: (tasks) => this.tasks.set(tasks),
      error: (err: HttpErrorResponse) => this.fail(err),
    });
  }

  private fail(err: HttpErrorResponse): void {
    this.error.set(
      err.status === 401 ? 'Sessão expirada. Faça login novamente.' : 'Erro ao processar tarefa.',
    );
  }

  isPending(id: number): boolean {
    return this.pendingTaskIds().has(id);
  }

  clearError(): void {
    this.error.set('');
  }

  private beginTaskAction(id: number): boolean {
    if (this.isPending(id)) {
      return false;
    }
    this.error.set('');
    this.pendingTaskIds.update((ids) => new Set(ids).add(id));
    return true;
  }

  private endTaskAction(id: number): void {
    this.pendingTaskIds.update((ids) => {
      const next = new Set(ids);
      next.delete(id);
      return next;
    });
  }
}
