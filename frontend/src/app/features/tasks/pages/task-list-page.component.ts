import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { STATUS_META, STATUSES, Task, TaskStatus } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { PageHeaderComponent } from '../../../shared/components/page-header.component';
import { TaskRowComponent } from '../components/task-row.component';

type Filter = 'all' | TaskStatus;

@Component({
  selector: 'app-task-list-page',
  imports: [RouterLink, PageHeaderComponent, TaskRowComponent],
  template: `
    <section class="page" animate.enter="page-enter">
      <app-page-header
        title="Tudo o que está em movimento."
        subtitle="Uma lista clara do que precisa de atenção."
        [mobileEyebrow]="'TAREFAS / ' + tasks().length"
        mobileTitle="Sua lista."
        [showMobileAction]="true"
      >
        <a class="btn btn-lime" routerLink="/tasks/new">
          <span class="desktop-label">Nova tarefa</span><span class="mobile-label">+ Nova tarefa</span>
          <span class="arrow" aria-hidden="true">↗</span>
        </a>
      </app-page-header>

      <div class="toolbar">
        <label class="search desktop-search">
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
        <label class="search mobile-search">
          <span aria-hidden="true">⌕</span>
          <span class="sr-only">Buscar tarefa</span>
          <input class="input" type="search" placeholder="Buscar tarefas" [value]="search()" (input)="onSearch($event)" />
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
        <div class="table desktop-table">
          <div class="table-head">
            <span>TAREFA</span>
            <span>ETAPA</span>
          </div>
          <ul>
            @for (task of filtered(); track task.id; let index = $index) {
              <li>
                <app-task-row [task]="task" [index]="index + 1" />
              </li>
            }
          </ul>
          <div class="table-foot">
            <span>01 — {{ filtered().length.toString().padStart(2, '0') }} / {{ filtered().length }} TAREFAS</span>
          </div>
        </div>
        <ul class="mobile-task-list">
          @for (task of filtered(); track task.id) {
            <li>
              <a [routerLink]="['/tasks', task.id]">
                <strong>{{ task.title }}</strong>
                <span class="mobile-status">{{ task.status === 'doing' ? 'EM CURSO' : task.status === 'done' ? 'FEITO' : 'A FAZER' }}</span>
              </a>
            </li>
          }
        </ul>
      }
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      app-page-header { margin-bottom: 49px; }
      .mobile-label, .mobile-search, .mobile-task-list { display: none; }
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
      .desktop-table { margin-top: 40px; }
      .table-foot { padding: 18px 4px; font-size: 11px; letter-spacing: 0.05em; color: var(--muted); }
      @media (max-width: 900px) {
        app-page-header { margin-bottom: 21px; }
        .desktop-label, .desktop-search, .filters, .desktop-table { display: none; }
        .mobile-label, .mobile-search { display: initial; }
        .header-action .btn, app-page-header .btn { width: 100%; background: var(--pine); color: var(--paper-raised); border-color: var(--pine); }
        .toolbar { display: block; }
        .search { flex-basis: auto; }
        .search .input { min-height: 48px; padding-left: 42px; font-size: 13px; }
        .mobile-task-list { display: block; list-style: none; margin: 33px 0 0; padding: 0; border-top: 1px solid var(--line); }
        .mobile-task-list li { border-bottom: 1px solid var(--line); }
        .mobile-task-list a { display: grid; gap: 8px; min-height: 82px; padding: 14px 3px; }
        .mobile-task-list strong { font-family: var(--font-display); font-size: 16px; color: var(--pine); }
        .mobile-status { font-size: 10px; color: var(--muted); }
      }
    `,
  ],
})
export class TaskListPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);
  private readonly route = inject(ActivatedRoute);

  readonly statuses = STATUSES;
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
    this.data
      .list()
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (tasks) => this.tasks.set(tasks),
        error: (err: HttpErrorResponse) =>
          this.error.set(apiErrorMessage(err, 'Não foi possível carregar as tarefas.')),
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
