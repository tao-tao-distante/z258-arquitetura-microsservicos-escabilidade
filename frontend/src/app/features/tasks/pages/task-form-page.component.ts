import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormsModule, NgForm } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { finalize } from 'rxjs';
import { apiErrorMessage } from '../../../core/http/http-error.utils';
import { STATUS_META, STATUSES, TaskPayload, TaskStatus } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';

@Component({
  selector: 'app-task-form-page',
  imports: [FormsModule, RouterLink],
  template: `
    <section class="page">
      <a class="link-arrow back" [routerLink]="cancelLink()">
        {{ isEdit() ? '← DETALHES' : '← TAREFAS' }}
      </a>

      <header>
        <h1 class="detail-title" [class.create]="!isEdit()">{{ heading() }}</h1>
      </header>

      <hr class="rule" />

      @if (error()) {
        <p class="error" role="alert" aria-live="assertive">{{ error() }}</p>
      }

      <form class="form" #form="ngForm" (ngSubmit)="submit(form)" novalidate>
        <label class="field">
          <span class="field-label">{{ isEdit() ? 'TÍTULO' : '01 / TÍTULO *' }}</span>
          <input
            class="input"
            name="title"
            [(ngModel)]="title"
            (ngModelChange)="clearError()"
            #titleModel="ngModel"
            required
            maxlength="160"
            placeholder="Ex.: Revisar contratos da API"
            aria-describedby="task-title-error"
          />
          @if ((titleModel.touched || submitted()) && titleModel.invalid) {
            <span id="task-title-error" class="error">Informe o título da tarefa.</span>
          }
        </label>

        <label class="field">
          <span class="field-label">{{ isEdit() ? 'DESCRIÇÃO' : '02 / DESCRIÇÃO' }}</span>
          <textarea
            class="input"
            name="description"
            [(ngModel)]="description"
            (ngModelChange)="clearError()"
            maxlength="500"
            rows="4"
            placeholder="Descreva os detalhes da tarefa..."
          ></textarea>
        </label>

        <div class="field">
          <label class="field-label" for="task-status">{{ isEdit() ? 'STATUS' : '03 / STATUS' }}</label>
          <select
            id="task-status"
            class="status-select"
            [class]="'status-select ' + statusValue"
            name="status"
            [(ngModel)]="statusValue"
            (ngModelChange)="clearError()"
          >
            @for (status of statuses; track status) {
              <option [value]="status">{{ meta(status).index }} / {{ meta(status).short }}</option>
            }
          </select>
        </div>

        <div class="actions">
          <a class="btn btn-outline" [routerLink]="cancelLink()">
            Cancelar <span class="arrow" aria-hidden="true">↗</span>
          </a>
          <button class="btn btn-primary" type="submit" [disabled]="loading()">
            {{ loading() ? 'Salvando...' : isEdit() ? 'Salvar alterações' : 'Criar tarefa' }}
            <span class="arrow" aria-hidden="true">↗</span>
          </button>
        </div>

        @if (!isEdit()) {
          <p class="note">POST /tasks · Status inicial sugerido: <code>todo</code></p>
        }
      </form>
    </section>
  `,
  styles: [
    `
      .page { display: grid; gap: 0; }
      .back { margin-bottom: 44px; font-size: 11px; letter-spacing: 0.05em; }
      header { margin-bottom: 64px; }
      .detail-title {
        font-size: clamp(2.4rem, 4vw, 50px);
        line-height: 1.02;
        letter-spacing: -0.035em;
      }
      .detail-title.create { max-width: 13ch; }
      .rule { margin-bottom: 28px; }
      .form { display: grid; gap: 0; max-width: 920px; }
      .field { gap: 12px; }
      .field + .field { margin-top: 38px; }
      .field-label { font-size: 11px; letter-spacing: 0.07em; }
      .actions { display: flex; justify-content: flex-end; gap: 10px; flex-wrap: wrap; margin-top: 26px; }
      .actions .btn { min-width: 155px; }
      .actions .btn-primary { min-width: 224px; }
      .status-select {
        justify-self: start;
        min-height: 28px;
        padding: 0 12px;
        border: 0;
        border-radius: 2px;
        appearance: none;
        font-family: var(--font-body);
        font-size: 10px;
        font-weight: 500;
        letter-spacing: 0.04em;
        color: var(--ink);
        text-transform: uppercase;
      }
      .status-select.todo { background: var(--band-todo); }
      .status-select.doing { background: var(--band-doing); color: var(--text-doing); }
      .status-select.done { background: var(--band-done); color: var(--text-done); }
      .note { margin-top: 38px; font-size: 11px; color: var(--muted); }
      .note code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
      @media (max-width: 900px) {
        .actions { flex-direction: column-reverse; }
        .actions .btn { width: 100%; }
      }
    `,
  ],
})
export class TaskFormPageComponent implements OnInit {
  private readonly data = inject(TaskDataService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly statuses = STATUSES;
  readonly loading = signal(false);
  readonly submitted = signal(false);
  readonly error = signal('');
  readonly taskId = signal<number | null>(null);
  readonly isEdit = computed(() => this.taskId() !== null);
  readonly heading = computed(() =>
    this.isEdit() ? 'Editar atividade.' : 'Uma nova coisa a ser feita.',
  );

  title = '';
  description = '';
  statusValue: TaskStatus = 'todo';

  ngOnInit(): void {
    const raw = this.route.snapshot.paramMap.get('id');
    if (raw === null) {
      return;
    }
    const id = Number(raw);
    if (!Number.isInteger(id) || id <= 0) {
      this.error.set('Tarefa inválida.');
      return;
    }
    this.taskId.set(id);
    this.loading.set(true);
    this.data.get(id).subscribe({
      next: (task) => {
        this.title = task.title;
        this.description = task.description;
        this.statusValue = task.status;
        this.loading.set(false);
      },
      error: (err: HttpErrorResponse) => {
        this.loading.set(false);
        this.error.set(apiErrorMessage(err, 'Não foi possível carregar a tarefa.'));
      },
    });
  }

  cancelLink(): string {
    const id = this.taskId();
    return id === null ? '/tasks' : `/tasks/${id}`;
  }

  meta(status: TaskStatus) {
    return STATUS_META[status];
  }

  clearError(): void {
    this.error.set('');
  }

  submit(form?: NgForm): void {
    this.submitted.set(true);
    const title = this.title.trim();
    if (form?.invalid || title.length === 0 || this.loading()) {
      form?.form.markAllAsTouched();
      this.error.set('Informe o título da tarefa.');
      return;
    }

    const payload: TaskPayload = {
      title,
      description: this.description.trim(),
      status: this.statusValue,
    };
    this.error.set('');
    this.loading.set(true);

    const id = this.taskId();
    const request = id === null ? this.data.create(payload) : this.data.update(id, payload);

    request.pipe(finalize(() => this.loading.set(false))).subscribe({
      next: (task) => void this.router.navigate(['/tasks', task.id]),
      error: (err: HttpErrorResponse) =>
        this.error.set(
          apiErrorMessage(err, this.isEdit() ? 'Falha ao salvar alterações.' : 'Falha ao criar tarefa.'),
        ),
    });
  }
}
