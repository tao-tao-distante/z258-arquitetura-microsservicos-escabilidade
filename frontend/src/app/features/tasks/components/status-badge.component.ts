import { Component, computed, input } from '@angular/core';
import { STATUS_META, TaskStatus } from '../models/task.model';

@Component({
  selector: 'app-status-badge',
  template: `<span [class]="'status-band ' + status()">{{ label() }}</span>`,
})
export class StatusBadgeComponent {
  readonly status = input.required<TaskStatus>();
  readonly withIndex = input(false);

  readonly label = computed(() => {
    const meta = STATUS_META[this.status()];
    return this.withIndex() ? `${meta.index} / ${meta.short}` : meta.short;
  });
}
