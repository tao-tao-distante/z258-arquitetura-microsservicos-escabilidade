import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { apiErrorMessage } from '../../core/http/http-error.utils';
import { Task, TaskStatus } from '../tasks/models/task.model';
import { TaskDataService } from '../tasks/services/task-data.service';
import { PageHeaderComponent } from '../../shared/components/page-header.component';
import { TaskRowComponent } from '../tasks/components/task-row.component';

@Component({
  selector: 'app-dashboard-page',
  imports: [RouterLink, PageHeaderComponent, TaskRowComponent],
  template: `
    <section class="page" animate.enter="page-enter">
      <app-page-header
        title="Menos ruído. Mais feito."
        subtitle="Sua semana em perspectiva. Pequenos avanços, todos os dias."
        mobileEyebrow="MEU WORKSPACE"
        mobileTitle="Hoje, com foco."
      >
        <a class="btn btn-lime" routerLink="/tasks/new">
          Nova tarefa <span class="arrow" aria-hidden="true">↗</span>
        </a>
      </app-page-header>

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }

      <div class="desktop-dashboard">
      <section class="hero">
        <div class="hero-copy">
          <span class="eyebrow">O QUE IMPORTA AGORA</span>
          <h2>Uma tarefa de cada vez.</h2>
          <p>O foco é terminar, não apenas começar.</p>
        </div>
        <div class="hero-progress">
          <div
            class="hero-bar"
            role="progressbar"
            [attr.aria-valuenow]="progress()"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-label="Percentual de tarefas concluídas"
          >
            <span [style.width.%]="progress()"></span>
          </div>
          <strong>{{ progress() }}%</strong>
          <span class="hero-label">CONCLUÍDO</span>
        </div>
      </section>

      <div class="stats">
        <a class="card stat" routerLink="/tasks">
          <strong>{{ tasks().length }}</strong>
          <span>TOTAL DE TAREFAS</span>
        </a>
        <a class="card stat" routerLink="/tasks" [queryParams]="{ status: 'todo' }">
          <strong>{{ count('todo') }}</strong>
          <span>POR COMEÇAR</span>
        </a>
        <a class="card stat" routerLink="/tasks" [queryParams]="{ status: 'doing' }">
          <strong>{{ count('doing') }}</strong>
          <span>EM ANDAMENTO</span>
        </a>
        <a class="card stat" routerLink="/tasks" [queryParams]="{ status: 'done' }">
          <strong>{{ count('done') }}</strong>
          <span>FINALIZADAS</span>
        </a>
      </div>

      <section class="ongoing">
        <div class="section-head">
          <h2>Em andamento</h2>
          <a class="link-arrow" routerLink="/tasks/board">VER TODAS ↗</a>
        </div>

        @if (ongoing().length === 0) {
          <div class="empty-state">
            <span class="eyebrow">NADA EM CURSO</span>
            <p class="muted">Mova uma tarefa para “Em curso” para vê-la aqui.</p>
            <a class="link-arrow" routerLink="/tasks">VER TODAS AS TAREFAS ↗</a>
          </div>
        } @else {
          <ul class="list">
            @for (task of ongoing(); track task.id; let index = $index) {
              <li>
                <app-task-row [task]="task" [index]="index + 1" />
              </li>
            }
          </ul>
        }
      </section>
      </div>

      <section class="mobile-dashboard">
        <div class="mobile-hero">
          <strong>{{ progress() }}%</strong>
          <span>DO TRABALHO CONCLUÍDO</span>
          <p class="mobile-progress">{{ count('done') }} concluídas de {{ tasks().length }} tarefas</p>
        </div>
        <span class="mobile-section-label">EM ANDAMENTO</span>
        <div class="mobile-tasks">
          @for (task of ongoing(); track task.id) {
            <a class="mobile-task" [routerLink]="['/tasks', task.id]">
              <strong>{{ task.title }}</strong>
              <span>{{ task.status === 'doing' ? 'EM CURSO' : 'A FAZER' }}</span>
            </a>
          }
        </div>
      </section>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      app-page-header { margin-bottom: 56px; }
      .mobile-dashboard { display: none; }
      .hero {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 40px;
        min-height: 218px;
        padding: 26px 30px 32px;
        background: var(--pine);
        color: var(--paper-raised);
        border-radius: 3px;
      }
      .hero-copy { display: grid; gap: 12px; }
      .hero-copy .eyebrow { color: var(--rail-text); font-size: 10px; }
      .hero-copy h2 {
        font-size: clamp(2rem, 3vw, 40px);
        letter-spacing: -0.03em;
        color: var(--paper-raised);
      }
      .hero-copy p { font-size: 14px; color: var(--rail-text); }
      .hero-progress { display: grid; gap: 8px; justify-items: end; min-width: 240px; }
      .hero-bar { width: 100%; height: 4px; background: rgba(221, 240, 138, 0.25); }
      .hero-bar span { display: block; height: 100%; background: var(--lime); }
      .hero-progress strong {
        font-family: var(--font-display);
        font-size: 38px;
        line-height: 1;
        color: var(--lime);
      }
      .hero-label { font-size: 10px; letter-spacing: 0.1em; color: var(--rail-text); }
      .stats { margin-top: 23px; display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 16px; }
      .stat {
        display: grid;
        gap: 14px;
        padding: 22px;
        min-height: 112px;
        align-content: space-between;
        transition: border-color 0.15s ease;
      }
      .stat:hover { border-color: var(--pine); }
      .stat strong {
        font-family: var(--font-display);
        font-size: 36px;
        line-height: 1;
        color: var(--pine);
      }
      .stat span { font-size: 10px; letter-spacing: 0.08em; color: var(--muted); }
      .ongoing { margin-top: 36px; }
      .section-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 4px; }
      .section-head h2 { font-size: 24px; }
      .list { list-style: none; margin: 0; padding: 0; border-top: 1px solid var(--line); }
      .list li { border-bottom: 1px solid var(--line); }
      @media (max-width: 900px) {
        app-page-header { margin-bottom: 38px; }
        .desktop-dashboard { display: none; }
        .mobile-dashboard { display: grid; }
        .mobile-hero { display: grid; min-height: 182px; padding: 23px 20px 20px; border-radius: 3px; background: var(--pine); color: var(--paper-raised); }
        .mobile-hero strong { font-family: var(--font-display); font-size: 58px; line-height: 1.1; color: var(--lime); }
        .mobile-hero > span { margin-top: 5px; font-size: 12px; }
        .mobile-progress { align-self: end; font-size: 12px; color: var(--rail-text); }
        .mobile-section-label { margin-top: 36px; font-size: 12px; color: var(--muted); }
        .mobile-tasks { display: grid; gap: 13px; margin-top: 29px; }
        .mobile-task { display: grid; gap: 9px; min-height: 78px; padding: 14px; border: 1px solid var(--line); border-radius: 3px; background: var(--paper-raised); }
        .mobile-task strong { font-family: var(--font-display); font-size: 16px; color: var(--pine); }
        .mobile-task span { font-size: 11px; color: var(--muted); }
      }
    `,
  ],
})
export class DashboardPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);

  readonly tasks = signal<Task[]>([]);
  readonly error = signal('');

  readonly progress = computed(() => {
    const total = this.tasks().length;
    return total === 0 ? 0 : Math.round((this.count('done') / total) * 100);
  });
  readonly ongoing = computed(() => {
    const doing = this.tasks().filter((task) => task.status === 'doing');
    const source = doing.length > 0 ? doing : this.tasks().filter((task) => task.status === 'todo');
    return source.slice(0, 3);
  });

  ngOnInit(): void {
    this.data.list().subscribe({
      next: (tasks) => this.tasks.set(tasks),
      error: (err: HttpErrorResponse) =>
        this.error.set(apiErrorMessage(err, 'Não foi possível carregar a visão geral.')),
    });
  }

  count(status: TaskStatus): number {
    return this.tasks().filter((task) => task.status === status).length;
  }
}
