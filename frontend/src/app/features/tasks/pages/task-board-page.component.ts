import { Component, computed, DestroyRef, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import {
  CdkDrag,
  CdkDragDrop,
  CdkDragPlaceholder,
  CdkDragPreview,
  CdkDropList,
  CdkDropListGroup,
} from '@angular/cdk/drag-drop';
import { CdkScrollable } from '@angular/cdk/scrolling';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { TaskCardComponent } from '../components/task-card.component';
import { DragTiltDirective } from '../drag/drag-tilt.directive';
import { STATUS_META, STATUSES, Task, TaskFeedback, TaskStatus } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { PageHeaderComponent } from '../../../shared/components/page-header.component';

const POSITION_STEP = 1024;
const FEEDBACK_MS = 700;

@Component({
  selector: 'app-task-board-page',
  imports: [
    RouterLink,
    TaskCardComponent,
    PageHeaderComponent,
    CdkDropListGroup,
    CdkDropList,
    CdkDrag,
    CdkDragPreview,
    CdkDragPlaceholder,
    CdkScrollable,
    DragTiltDirective,
  ],
  template: `
    <section class="page" animate.enter="page-enter">
      <app-page-header
        title="Do plano à entrega."
        subtitle="Arraste o trabalho de uma etapa para outra."
        mobileEyebrow="QUADRO"
        mobileTitle="Do plano à entrega."
        [showMobileAction]="true"
      >
        <a class="btn btn-lime" routerLink="/tasks/new">
          Nova tarefa <span class="arrow" aria-hidden="true">↗</span>
        </a>
      </app-page-header>

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }

      <div class="board" cdkDropListGroup cdkScrollable>
        @for (status of statuses; track status) {
          <section class="column" [attr.aria-label]="meta(status).label">
            <div class="column-head" [class]="'column-head ' + status">
              <span class="eyebrow">{{ meta(status).index }} / {{ meta(status).short }}</span>
              <span class="count">{{ byStatus(status).length.toString().padStart(2, '0') }}</span>
            </div>
            <div
              class="column-body"
              cdkDropList
              [cdkDropListData]="status"
              (cdkDropListDropped)="drop($event)"
            >
              @for (task of byStatus(status); track task.id) {
                <app-task-card
                  cdkDrag
                  [cdkDragData]="task"
                  [cdkDragDisabled]="isPending(task.id)"
                  [cdkDragStartDelay]="dragDelay"
                  [task]="task"
                  [feedback]="feedback()[task.id] ?? null"
                  (move)="moveTo(task, $event)"
                >
                  <ng-template cdkDragPreview [matchSize]="true">
                    <div class="drag-preview">
                      <div class="tilt" appDragTilt>
                        <app-task-card [task]="task" />
                      </div>
                    </div>
                  </ng-template>
                  <ng-template cdkDragPlaceholder>
                    <div class="drop-slot"></div>
                  </ng-template>
                </app-task-card>
              }
              @if (byStatus(status).length === 0) {
                <p class="empty">Nenhuma tarefa neste estágio.</p>
              }
            </div>
          </section>
        }
      </div>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      app-page-header { margin-bottom: 57px; }
      .board { display: grid; grid-template-columns: repeat(3, minmax(0, 336px)); gap: 24px; align-items: stretch; }
      .column { display: grid; grid-template-rows: auto 1fr; }
      .column-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        min-height: 58px;
        padding: 12px 17px;
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
        min-height: 515px;
        align-content: start;
        transition: background-color 150ms ease, border-color 150ms ease;
      }
      .column-body.cdk-drop-list-receiving { border-color: var(--rail-muted); }
      .column-body.cdk-drop-list-dragging {
        border-color: var(--pine);
        background: color-mix(in srgb, var(--lime) 22%, var(--paper-raised));
      }
      .column-body.cdk-drop-list-dragging .empty { display: none; }
      .column-body.cdk-drop-list-dragging app-task-card:not(.cdk-drag-placeholder) {
        transition: transform 220ms cubic-bezier(0.2, 0.8, 0.2, 1);
      }
      .drop-slot {
        min-height: 138px;
        border: 1px dashed var(--pine);
        border-radius: 3px;
        background: color-mix(in srgb, var(--lime) 35%, transparent);
        opacity: 0.45;
      }
      .drag-preview {
        overflow: visible;
        padding: 0;
        border: 0;
        background: transparent;
        color: inherit;
        cursor: grabbing;
      }
      .drag-preview.cdk-drag-animating { transition: transform 260ms cubic-bezier(0.2, 0.8, 0.2, 1); }
      .tilt {
        border-radius: 3px;
        box-shadow: 0 22px 40px -14px rgb(24 61 53 / 0.38), 0 6px 14px -6px rgb(24 61 53 / 0.18);
        will-change: transform;
      }
      .tilt app-task-card { cursor: grabbing; }
      app-task-card.cdk-drag-animating { transition: transform 260ms cubic-bezier(0.2, 0.8, 0.2, 1); }
      .empty { padding: 16px 4px; font-size: 12px; color: var(--muted); }
      @media (max-width: 900px) {
        app-page-header { margin-bottom: 24px; }
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
  private readonly timers = new Set<ReturnType<typeof setTimeout>>();
  private readonly pending = signal<ReadonlySet<number>>(new Set());

  readonly statuses = STATUSES;
  readonly dragDelay = { touch: 200, mouse: 0 };
  readonly tasks = signal<Task[]>([]);
  readonly error = signal('');
  readonly feedback = signal<Partial<Record<number, TaskFeedback>>>({});

  private readonly grouped = computed(() => {
    const map: Record<TaskStatus, Task[]> = { todo: [], doing: [], done: [] };
    for (const task of this.tasks()) {
      map[task.status].push(task);
    }
    for (const status of STATUSES) {
      map[status].sort((a, b) => a.position - b.position || b.id - a.id);
    }
    return map;
  });

  constructor() {
    inject(DestroyRef).onDestroy(() => this.timers.forEach((timer) => clearTimeout(timer)));
  }

  ngOnInit(): void {
    this.load();
  }

  byStatus(status: TaskStatus): Task[] {
    return this.grouped()[status];
  }

  meta(status: TaskStatus) {
    return STATUS_META[status];
  }

  isPending(id: number): boolean {
    return this.pending().has(id);
  }

  drop(event: CdkDragDrop<TaskStatus, TaskStatus, Task>): void {
    const status = event.container.data;
    if (event.previousContainer.data === status && event.previousIndex === event.currentIndex) {
      return;
    }
    const task = event.item.data;
    const column = this.byStatus(status).filter((item) => item.id !== task.id);
    this.commit(task, status, column[event.currentIndex - 1] ?? null, column[event.currentIndex] ?? null);
  }

  moveTo(task: Task, status: TaskStatus): void {
    if (task.status === status) {
      return;
    }
    this.commit(task, status, null, this.byStatus(status)[0] ?? null);
  }

  private load(): void {
    this.data.list().subscribe({
      next: (tasks) => this.tasks.set(tasks),
      error: (err: HttpErrorResponse) =>
        this.error.set(apiErrorMessage(err, 'Não foi possível carregar o quadro.')),
    });
  }

  private commit(task: Task, status: TaskStatus, before: Task | null, after: Task | null): void {
    this.replace({ ...task, status, position: this.localPosition(before, after) });
    this.setPending(task.id, true);
    this.error.set('');
    this.data.move(task.id, { status, before_id: before?.id ?? null, after_id: after?.id ?? null }).subscribe({
      next: (saved) => {
        this.replace(saved);
        this.settle(task.id, 'saved');
      },
      error: (err: HttpErrorResponse) => {
        if (err.status === 409) {
          this.setPending(task.id, false);
          this.error.set('O quadro mudou em outro lugar. Recarregamos as tarefas.');
          this.load();
          return;
        }
        this.replace(task);
        this.settle(task.id, 'failed');
        this.error.set(apiErrorMessage(err, 'Não foi possível mover a tarefa.'));
      },
    });
  }

  private localPosition(before: Task | null, after: Task | null): number {
    if (before && after) {
      return (before.position + after.position) / 2;
    }
    if (before) {
      return before.position + POSITION_STEP;
    }
    return after ? after.position - POSITION_STEP : 0;
  }

  private replace(updated: Task): void {
    this.tasks.update((tasks) => tasks.map((item) => (item.id === updated.id ? updated : item)));
  }

  private setPending(id: number, pending: boolean): void {
    this.pending.update((current) => {
      const next = new Set(current);
      if (pending) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }

  private settle(id: number, result: TaskFeedback): void {
    this.setPending(id, false);
    this.feedback.update((current) => ({ ...current, [id]: result }));
    const timer = setTimeout(() => {
      this.timers.delete(timer);
      this.feedback.update((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
    }, FEEDBACK_MS);
    this.timers.add(timer);
  }
}
