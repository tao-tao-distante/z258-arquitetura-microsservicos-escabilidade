import { Component, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { DueDateComponent } from '../components/due-date.component';
import { PriorityBadgeComponent } from '../components/priority-badge.component';
import { StatusBadgeComponent } from '../components/status-badge.component';
import { TaskHistoryComponent } from '../components/task-history.component';
import { STATUS_META, Task, TaskStatus } from '../models/task.model';
import { formatDate, formatDateTime } from '../task.utils';
import { TaskDataService } from '../services/task-data.service';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog.component';

@Component({
  selector: 'app-task-detail-page',
  imports: [
    RouterLink,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    DueDateComponent,
    TaskHistoryComponent,
    ConfirmDialogComponent,
  ],
  template: `
    <section class="page" animate.enter="page-enter">
      <a class="link-arrow back" routerLink="/tasks">← VOLTAR ÀS TAREFAS</a>

      @if (loading()) {
        <p class="muted" role="status">Carregando tarefa…</p>
      } @else if (task(); as current) {
        @if (error()) {
          <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
        }

        <header class="head">
          <h1 class="detail-title">{{ current.title }}</h1>
          <app-status-badge [status]="current.status" [withIndex]="true" />
        </header>

        <hr class="rule" />

        <section class="description">
          <span class="eyebrow">DESCRIÇÃO</span>
          @if (current.description) {
            <p>{{ current.description }}</p>
          } @else {
            <p class="muted">Esta tarefa ainda não tem descrição.</p>
          }
        </section>

        <hr class="rule" />

        <dl class="meta">
          <div>
            <dt>STATUS</dt>
            <dd>{{ meta(current.status).label }}</dd>
          </div>
          <div>
            <dt>PRIORIDADE</dt>
            <dd><app-priority-badge [priority]="current.priority" /></dd>
          </div>
          <div>
            <dt>PRAZO</dt>
            <dd class="due">
              @if (current.due_date) {
                <span>{{ date(current.due_date) }}</span>
                <app-due-date [task]="current" />
              } @else {
                <span class="muted">Sem prazo</span>
              }
            </dd>
          </div>
          <div>
            <dt>CRIADA EM</dt>
            <dd>{{ dateTime(current.created_at) }}</dd>
          </div>
          <div>
            <dt>IDENTIFICADOR</dt>
            <dd>#{{ current.id }}</dd>
          </div>
        </dl>

        <hr class="rule history-rule" />

        <app-task-history [taskId]="current.id" />

        <div class="actions">
          <a class="btn btn-primary" [routerLink]="['/tasks', current.id, 'edit']">
            Editar tarefa <span class="arrow" aria-hidden="true">↗</span>
          </a>
          <button class="btn btn-danger" type="button" (click)="confirmOpen.set(true)">
            Excluir <span class="arrow" aria-hidden="true">↗</span>
          </button>
        </div>

        @if (confirmOpen()) {
          <app-confirm-dialog
            title="Excluir esta tarefa?"
            description="Essa ação não pode ser desfeita. A tarefa será removida permanentemente do workspace."
            confirmLabel="Sim, excluir"
            [pending]="pending()"
            (cancel)="confirmOpen.set(false)"
            (confirm)="remove()"
          />
        }
      } @else {
        <div class="empty-state">
          <span class="eyebrow">NÃO ENCONTRADA</span>
          <p class="muted">{{ error() || 'Esta tarefa não existe ou foi removida.' }}</p>
          <a class="link-arrow" routerLink="/tasks">VOLTAR PARA TAREFAS ↗</a>
        </div>
      }
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      .back { margin-bottom: 57px; font-size: 12px; letter-spacing: 0.02em; }
      .head {
        display: grid;
        grid-template-columns: 1fr auto;
        align-items: start;
        gap: 32px;
        margin-bottom: 52px;
      }
      .detail-title {
        font-size: clamp(2.6rem, 4.4vw, 54px);
        line-height: 1.02;
        letter-spacing: -0.035em;
        max-width: 14ch;
      }
      .head app-status-badge { margin-top: 10px; }
      .head + .rule { margin-bottom: 46px; }
      .description { display: grid; gap: 18px; margin-bottom: 63px; }
      .description p { font-size: 19px; line-height: 1.32; max-width: 62ch; white-space: pre-wrap; }
      .description + .rule { margin-bottom: 31px; }
      .meta { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 32px 24px; margin: 0; }
      dd.due { display: flex; flex-wrap: wrap; align-items: baseline; gap: 10px; }
      .history-rule { margin: 52px 0 31px; }
      dt { font-size: 10px; letter-spacing: 0.1em; color: var(--muted); margin-bottom: 10px; }
      dd { margin: 0; font-size: 16px; color: var(--ink); }
      .actions { display: flex; gap: 14px; flex-wrap: wrap; margin-top: 92px; }
      .actions .btn-primary { min-width: 194px; }
      @media (max-width: 900px) {
        .head { grid-template-columns: 1fr; gap: 14px; }
        .head app-status-badge { margin-top: 0; justify-self: start; }
        .meta { grid-template-columns: 1fr; gap: 18px; }
        .actions { margin-top: 36px; }
        .actions .btn { flex: 1; min-width: 0; }
      }
    `,
  ],
})
export class TaskDetailPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly task = signal<Task | null>(null);
  readonly loading = signal(true);
  readonly pending = signal(false);
  readonly error = signal('');
  readonly confirmOpen = signal(false);

  ngOnInit(): void {
    const id = Number(this.route.snapshot.paramMap.get('id'));
    if (!Number.isInteger(id) || id <= 0) {
      this.loading.set(false);
      this.error.set('Tarefa inválida.');
      return;
    }
    this.data
      .get(id)
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (task) => this.task.set(task),
        error: (err: HttpErrorResponse) =>
          this.error.set(apiErrorMessage(err, 'Não foi possível carregar a tarefa.')),
      });
  }

  meta(status: TaskStatus) {
    return STATUS_META[status];
  }

  date(iso: string): string {
    return formatDate(iso);
  }

  dateTime(iso: string): string {
    return formatDateTime(iso);
  }

  remove(): void {
    const current = this.task();
    if (!current || this.pending()) {
      return;
    }
    this.pending.set(true);
    this.error.set('');
    this.data
      .remove(current.id)
      .pipe(finalize(() => this.pending.set(false)))
      .subscribe({
        next: () => void this.router.navigate(['/tasks']),
        error: (err: HttpErrorResponse) => {
          this.confirmOpen.set(false);
          this.error.set(apiErrorMessage(err, 'Não foi possível excluir a tarefa.'));
        },
      });
  }

}
