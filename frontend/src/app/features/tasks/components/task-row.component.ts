import { Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Task } from '../models/task.model';
import { StatusBadgeComponent } from './status-badge.component';

@Component({
  selector: 'app-task-row',
  imports: [RouterLink, StatusBadgeComponent],
  template: `
    <a class="task-row" [routerLink]="['/tasks', task().id]">
      <span class="index">{{ index().toString().padStart(2, '0') }}</span>
      <span class="copy">
        <span class="title">{{ task().title }}</span>
        @if (task().description) { <span class="desc">{{ task().description }}</span> }
      </span>
      <app-status-badge [status]="task().status" [withIndex]="true" />
      <span class="arrow" aria-hidden="true">↗</span>
    </a>
  `,
  styles: [`
    :host { display: block; }
    .task-row { display: grid; grid-template-columns: 44px 1fr 280px 24px; align-items: center; gap: 16px; padding: 23px 4px; }
    .task-row:hover .title { text-decoration: underline; text-underline-offset: 3px; }
    .index { font-size: 11px; color: var(--muted); font-variant-numeric: tabular-nums; }
    .copy { display: grid; gap: 3px; min-width: 0; }
    .title { font-family: var(--font-display); font-weight: 700; font-size: 18px; color: var(--pine); }
    .desc { font-size: 12px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .arrow { font-size: 20px; color: var(--pine); justify-self: end; }
  `],
})
export class TaskRowComponent {
  readonly task = input.required<Task>();
  readonly index = input.required<number>();
}
