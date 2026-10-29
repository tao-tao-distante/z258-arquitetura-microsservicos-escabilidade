import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { StatusBadgeComponent } from '../components/status-badge.component';
import { STATUS_META, Task, TaskStatus } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';

type Filter = 'all' | TaskStatus;

@Component({
  selector: 'app-task-list-page',
  imports: [RouterLink, StatusBadgeComponent],
  template: `
    <section class="page">
      <header class="head">
        <div>
          <h1 class="page-title">Tudo o que está em movimento.</h1>
          <p class="page-subtitle">Uma lista clara do que precisa de atenção.</p>
        </div>
        <a class="btn btn-lime" routerLink="/tasks/new">
          Nova tarefa <span class="arrow" aria-hidden="true">↗</span>
        </a>
      </header>

      <div class="toolbar">
        <label class="search">
          <span aria-hidden="true">⌕</span>
          <span class="sr-only">Buscar tarefa</span>
          <input
            class="input"
            type="search"
            placeholder="Buscar por título ou descrição"
            [value]="search()"
            (input)="onSearch($event)"
          />
        </label>

        <div class="filters" role="group" aria-label="Filtrar por status">
          <button
            type="button"
            [class.active]="filter() === 'all'"
            [attr.aria-pressed]="filter() === 'all'"
            (click)="setFilter('all')"
          >TODAS <span>{{ tasks().length }}</span></button>
          @for (status of statuses; track status) {
            <button
              type="button"
              [class.active]="filter() === status"
              [attr.aria-pressed]="filter() === status"
              (click)="setFilter(status)"
            >{{ filterLabel(status) }} <span>{{ count(status) }}</span></button>
          }
        </div>
      </div>

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }

      @if (loading()) {
        <p class="muted" role="status">Carregando tarefas…</p>
      } @else if (filtered().length === 0) {
        <div class="empty-state">
          <span class="eyebrow">LISTA VAZIA</span>
          <p class="muted">Nenhuma tarefa corresponde a este filtro.</p>
          <a class="link-arrow" routerLink="/tasks/new">CRIAR PRIMEIRA TAREFA ↗</a>
        </div>
      } @else {
        <div class="table">
          <div class="table-head">
            <span>TAREFA</span>
            <span>ETAPA</span>
          </div>
          <ul>
            @for (task of filtered(); track task.id; let index = $index) {
              <li>
                <a [routerLink]="['/tasks', task.id]">
                  <span class="index">{{ (index + 1).toString().padStart(2, '0') }}</span>
                  <span class="copy">
                    <span class="title">{{ task.title }}</span>
                    @if (task.description) {
                      <span class="desc">{{ task.description }}</span>
                    }
                  </span>
                  <app-status-badge [status]="task.status" [withIndex]="true" />
                  <span class="arrow" aria-hidden="true">↗</span>
                </a>
              </li>
            }
          </ul>
          <div class="table-foot">
            <span>01 — {{ filtered().length.toString().padStart(2, '0') }} / {{ filtered().length }} TAREFAS</span>
          </div>
        </div>
      }
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 26px; }
      .head { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; }
      .toolbar {
        display: flex;
        align-items: center;
        justify-content: flex-start;
        gap: 30px;
        flex-wrap: wrap;
      }
      .search { position: relative; flex: 0 1 464px; }
      .search > span[aria-hidden] {
        position: absolute;
        left: 16px;
        top: 50%;
        transform: translateY(-50%);
        font-size: 18px;
        color: var(--muted);
      }
      .search .input { min-height: 50px; padding-left: 44px; font-size: 12.5px; }
      .filters { display: flex; flex-wrap: wrap; gap: 56px; }
      .filters button {
        border: 0;
        background: none;
        padding: 4px 0;
        font-size: 11px;
        font-weight: 500;
        letter-spacing: 0.07em;
        color: var(--muted);
      }
      .filters button span { margin-left: 5px; font-variant-numeric: tabular-nums; }
      .filters button:hover { color: var(--pine); }
      .filters button.active { color: var(--pine); }
      .table { display: grid; }
      .table-head {
        display: grid;
        grid-template-columns: 44px 1fr 280px 24px;
        gap: 16px;
        padding: 0 4px 14px;
        border-bottom: 1px solid var(--line);
        font-size: 10px;
        letter-spacing: 0.1em;
        color: var(--muted);
      }
      .table-head span:first-child { grid-column: 1 / 3; }
      .table-head span:last-child { grid-column: 3; }
      ul { list-style: none; margin: 0; padding: 0; }
      li { border-bottom: 1px solid var(--line); }
      li a {
        display: grid;
        grid-template-columns: 44px 1fr 280px 24px;
        align-items: center;
        gap: 16px;
        padding: 24px 4px;
      }
      li a:hover .title { text-decoration: underline; text-underline-offset: 3px; }
      .index { font-size: 11px; color: var(--muted); font-variant-numeric: tabular-nums; }
      .copy { display: grid; gap: 3px; min-width: 0; }
      .title { font-family: var(--font-display); font-weight: 600; font-size: 16px; color: var(--pine); }
      .desc {
        font-size: 12px;
        color: var(--muted);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .arrow { font-size: 16px; color: var(--pine); justify-self: end; }
      .table-foot { padding: 18px 4px; font-size: 11px; letter-spacing: 0.05em; color: var(--muted); }
      @media (max-width: 900px) {
        .head { flex-direction: column; }
        .toolbar { flex-direction: column; align-items: stretch; gap: 16px; }
        .search { flex-basis: auto; }
        .filters { justify-content: space-between; gap: 10px; }
        .table-head { display: none; }
        li a { grid-template-columns: 30px 1fr 20px; gap: 12px; }
        app-status-badge { display: none; }
      }
    `,
  ],
})
export class TaskListPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);
  private readonly route = inject(ActivatedRoute);

  readonly statuses: TaskStatus[] = ['todo', 'doing', 'done'];
  readonly tasks = signal<Task[]>([]);
  readonly search = signal('');
  readonly filter = signal<Filter>('all');
  readonly loading = signal(true);
  readonly error = signal('');

  readonly filtered = computed(() => {
    const term = this.search().trim().toLowerCase();
    const status = this.filter();
    return this.tasks().filter((task) => {
      const matchesStatus = status === 'all' || task.status === status;
      const matchesTerm =
        term.length === 0 ||
        task.title.toLowerCase().includes(term) ||
        task.description.toLowerCase().includes(term);
      return matchesStatus && matchesTerm;
    });
  });

  ngOnInit(): void {
    const status = this.route.snapshot.queryParamMap.get('status');
    if (status === 'todo' || status === 'doing' || status === 'done') {
      this.filter.set(status);
    }
    this.data.list().subscribe({
      next: (tasks) => {
        this.tasks.set(tasks);
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.loading.set(false);
        this.error.set(apiErrorMessage(err, 'Não foi possível carregar as tarefas.'));
      },
    });
  }

  onSearch(event: Event): void {
    this.search.set((event.target as HTMLInputElement).value);
  }

  setFilter(filter: Filter): void {
    this.filter.set(filter);
  }

  count(status: TaskStatus): number {
    return this.tasks().filter((task) => task.status === status).length;
  }

  filterLabel(status: TaskStatus): string {
    return status === 'done' ? 'FEITAS' : STATUS_META[status].short;
  }
}
