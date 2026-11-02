import { Component, inject, input, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { finalize } from 'rxjs';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { FieldChange, HistoryAction, TaskHistoryEntry } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { ACTION_LABELS, describeChange, formatDateTime } from '../task.utils';

@Component({
  selector: 'app-task-history',
  template: `
    <section class="history" aria-labelledby="task-history-title">
      <span class="eyebrow" id="task-history-title">HISTÓRICO DE ALTERAÇÕES</span>

      @if (error()) {
        <p class="error" role="alert">{{ error() }}</p>
      }

      @if (entries().length === 0) {
        @if (loading()) {
          <p class="muted" role="status">Carregando histórico…</p>
        } @else if (!error()) {
          <p class="muted">Nenhuma alteração registrada.</p>
        }
      } @else {
        <ol class="timeline">
          @for (entry of entries(); track entry.id) {
            <li [class]="'entry ' + entry.action">
              <p class="when">
                <time [attr.datetime]="entry.changed_at">{{ when(entry.changed_at) }}</time>
                <span aria-hidden="true">·</span>
                <strong>{{ actionLabel(entry.action) }}</strong>
              </p>
              @if (entry.changes.length > 0) {
                <ul class="changes">
                  @for (change of entry.changes; track $index) {
                    <li>{{ describe(change, entry.action) }}</li>
                  }
                </ul>
              }
            </li>
          }
        </ol>
      }

      @if (hasMore()) {
        <button class="link-arrow more" type="button" (click)="loadMore()" [disabled]="loading()">
          {{ loading() ? 'CARREGANDO…' : 'CARREGAR MAIS ↓' }}
        </button>
      }
    </section>
  `,
  styles: [
    `
      .history { display: grid; gap: 22px; }
      .timeline { list-style: none; margin: 0; padding: 0; border-left: 1px solid var(--line); }
      .entry { position: relative; padding: 0 0 24px 22px; }
      .entry:last-child { padding-bottom: 0; }
      .entry::before {
        content: '';
        position: absolute;
        left: -5px;
        top: 4px;
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: var(--paper);
        border: 1px solid var(--pine);
      }
      .entry.created::before { background: var(--lime); }
      .entry.moved::before { background: var(--pine); }
      .when { display: flex; flex-wrap: wrap; gap: 8px; align-items: baseline; font-size: 12px; color: var(--muted); }
      .when time { font-variant-numeric: tabular-nums; }
      .when strong { font-size: 13px; font-weight: 600; color: var(--ink); }
      .changes { list-style: none; margin: 8px 0 0; padding: 0; display: grid; gap: 4px; font-size: 14px; color: var(--ink); }
      .more { justify-self: start; border: 0; background: none; padding: 0; font-size: 11px; letter-spacing: 0.06em; cursor: pointer; }
      .more:disabled { cursor: progress; opacity: 0.6; }
    `,
  ],
})
export class TaskHistoryComponent implements OnInit {
  private readonly data = inject(TaskDataService);

  readonly taskId = input.required<number>();

  readonly entries = signal<TaskHistoryEntry[]>([]);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly hasMore = signal(false);

  ngOnInit(): void {
    this.load();
  }

  loadMore(): void {
    if (!this.loading()) {
      this.load();
    }
  }

  when(iso: string): string {
    return formatDateTime(iso);
  }

  actionLabel(action: HistoryAction): string {
    return ACTION_LABELS[action] ?? action;
  }

  describe(change: FieldChange, action: HistoryAction): string {
    return describeChange(change, action);
  }

  private load(): void {
    const pageSize = TaskDataService.historyPageSize;
    this.loading.set(true);
    this.error.set('');
    this.data
      .history(this.taskId(), this.entries().length, pageSize)
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (page) => {
          this.entries.update((current) => [...current, ...page]);
          this.hasMore.set(page.length === pageSize);
        },
        error: (err: HttpErrorResponse) =>
          this.error.set(apiErrorMessage(err, 'Não foi possível carregar o histórico.')),
      });
  }
}
