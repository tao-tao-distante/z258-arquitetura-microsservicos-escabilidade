import { Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Task } from '../models/task.model';

@Component({
  selector: 'app-task-card',
  imports: [RouterLink],
  template: `
    <article class="task">
      <h3>{{ task().title }}</h3>
      <p class="meta">Tarefa #{{ task().id.toString().padStart(3, '0') }}</p>
      <hr />
      <a class="link-arrow" [routerLink]="['/tasks', task().id]">
        ABRIR DETALHES <span aria-hidden="true">↗</span>
      </a>
    </article>
  `,
  styles: [
    `
      .task {
        display: grid;
        gap: 10px;
        min-height: 138px;
        padding: 16px 14px 13px;
        background: var(--paper);
        border: 1px solid var(--line);
        border-radius: 3px;
        transition: border-color 0.15s ease;
      }
      .task:hover { border-color: var(--pine); }
      h3 { font-size: 17px; line-height: 1.25; }
      .meta { font-size: 12px; color: var(--muted); }
      hr { border: 0; height: 1px; background: var(--line); margin: 2px 0; }
      .link-arrow { font-size: 10px; letter-spacing: 0.06em; }
    `,
  ],
})
export class TaskCardComponent {
  readonly task = input.required<Task>();
}
