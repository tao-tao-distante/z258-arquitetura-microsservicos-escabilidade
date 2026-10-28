import { Component, input, output, signal } from '@angular/core';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';
import { Task, TaskStatus } from '../models/task.model';

@Component({
  selector: 'app-task-card',
  imports: [StatusBadgeComponent],
  template: `
    <article class="card task" [attr.aria-busy]="pending()">
      <div class="row" style="justify-content: space-between; align-items: flex-start;">
        <h3>{{ task().title }}</h3>
        <app-status-badge [status]="task().status" />
      </div>
      @if (task().description) {
        <p class="muted">{{ task().description }}</p>
      }
      <div class="row actions">
        @if (task().status !== 'done') {
          <button type="button" (click)="advance.emit(task())" [disabled]="pending() || confirmingDelete()">
            {{ task().status === 'todo' ? 'Iniciar' : 'Concluir' }}
          </button>
        }
        <button
          class="danger"
          type="button"
          (click)="confirmingDelete.set(true)"
          [disabled]="pending() || confirmingDelete()"
          [attr.aria-label]="'Excluir tarefa ' + task().title"
        >Excluir</button>
      </div>
      @if (confirmingDelete()) {
        <div class="confirm stack" role="alertdialog" aria-live="assertive" aria-label="Confirmar exclusão">
          <span>Excluir “{{ task().title }}” permanentemente?</span>
          <div class="row">
            <button type="button" class="danger" (click)="confirmRemove()" [disabled]="pending()">Confirmar</button>
            <button type="button" (click)="confirmingDelete.set(false)" [disabled]="pending()">Cancelar</button>
          </div>
        </div>
      }
    </article>
  `,
  styles: [
    `
      .task { display: grid; gap: 0.6rem; }
      h3 { margin: 0; font-size: 1rem; }
      p { margin: 0; font-size: 0.9rem; }
      .actions { margin-top: 0.25rem; }
      .actions button { font-size: 0.8rem; padding: 0.4rem 0.7rem; }
      .confirm { border-top: 1px solid var(--border); padding-top: 0.75rem; font-size: 0.85rem; }
    `,
  ],
})
export class TaskCardComponent {
  readonly task = input.required<Task>();
  readonly pending = input(false);
  readonly advance = output<Task>();
  readonly remove = output<Task>();
  readonly confirmingDelete = signal(false);

  confirmRemove(): void {
    if (!this.pending()) {
      this.remove.emit(this.task());
    }
  }
}

export type { TaskStatus };
