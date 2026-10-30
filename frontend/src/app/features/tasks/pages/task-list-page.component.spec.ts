import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { Task } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { todayISO } from '../task.utils';
import { TaskListPageComponent } from './task-list-page.component';

function shiftDays(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() + days);
  return todayISO(date);
}

const base: Task = {
  id: 1,
  user_id: 1,
  title: 'Task',
  description: '',
  status: 'todo',
  priority: 'medium',
  due_date: null,
  position: 0,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T10:00:00Z',
};

const tasks: Task[] = [
  { ...base, id: 1, title: 'Baixa sem prazo', priority: 'low', created_at: '2026-10-03T10:00:00Z' },
  { ...base, id: 2, title: 'Urgente atrasada', priority: 'urgent', due_date: shiftDays(-2), status: 'doing' },
  { ...base, id: 3, title: 'Alta para semana', priority: 'high', due_date: shiftDays(6), created_at: '2026-10-05T10:00:00Z' },
  { ...base, id: 4, title: 'Feita vencida', priority: 'high', due_date: shiftDays(-5), status: 'done' },
];

function setup(query: Record<string, string> = {}) {
  TestBed.configureTestingModule({
    imports: [TaskListPageComponent],
    providers: [
      provideRouter([]),
      { provide: TaskDataService, useValue: { list: vi.fn(() => of(tasks)) } },
      { provide: ActivatedRoute, useValue: { snapshot: { queryParamMap: convertToParamMap(query) } } },
    ],
  });
  const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const fixture = TestBed.createComponent(TaskListPageComponent);
  fixture.detectChanges();
  return { fixture, component: fixture.componentInstance, navigate };
}

const ids = (list: Task[]) => list.map((task) => task.id);

describe('TaskListPageComponent', () => {
  it('renders priority, due date and creation columns', () => {
    const { fixture } = setup();
    const element = fixture.nativeElement as HTMLElement;
    const headers = Array.from(element.querySelectorAll('.table-head span')).map((span) => span.textContent?.trim());

    expect(headers).toEqual(['TAREFA', 'PRIORIDADE', 'PRAZO', 'CRIADA EM', 'ETAPA']);
    expect(element.querySelector('app-task-row .priority.urgent')).not.toBeNull();
    expect(element.querySelector('app-task-row .due.overdue')?.textContent).toContain('Atrasada há 2 dias');
  });

  it('filters by priority and syncs the URL', () => {
    const { fixture, component, navigate } = setup();
    const select = (fixture.nativeElement as HTMLElement).querySelector('#priority-filter') as HTMLSelectElement;

    select.value = 'high';
    select.dispatchEvent(new Event('change'));

    expect(ids(component.filtered())).toEqual([3, 4]);
    expect(navigate).toHaveBeenLastCalledWith([], expect.objectContaining({
      queryParams: { status: null, priority: 'high', overdue: null, sort: null },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    }));
  });

  it('shows only open overdue tasks', () => {
    const { fixture, component } = setup();
    const toggle = (fixture.nativeElement as HTMLElement).querySelector('.overdue-toggle') as HTMLButtonElement;

    expect(toggle.textContent).toContain('1');
    toggle.click();
    fixture.detectChanges();

    expect(ids(component.filtered())).toEqual([2]);
    expect(toggle.getAttribute('aria-pressed')).toBe('true');
  });

  it.each([
    { sort: 'position', want: [1, 2, 3, 4] },
    { sort: 'priority', want: [2, 4, 3, 1] },
    { sort: 'created_at', want: [3, 1, 4, 2] },
    { sort: 'due_date', want: [4, 2, 3, 1] },
  ])('sorts by $sort', ({ sort, want }) => {
    const { fixture, component } = setup();
    const select = (fixture.nativeElement as HTMLElement).querySelector('#task-sort') as HTMLSelectElement;

    select.value = sort;
    select.dispatchEvent(new Event('change'));

    expect(ids(component.filtered())).toEqual(want);
  });

  it('restores filters and sort from the URL', () => {
    const { component } = setup({ status: 'doing', priority: 'urgent', overdue: 'true', sort: 'due_date' });

    expect(component.filter()).toBe('doing');
    expect(component.priorityFilter()).toBe('urgent');
    expect(component.overdueOnly()).toBe(true);
    expect(component.sort()).toBe('due_date');
    expect(ids(component.filtered())).toEqual([2]);
  });

  it('ignores unknown values in the URL', () => {
    const { component } = setup({ priority: 'critical', sort: 'title' });

    expect(component.priorityFilter()).toBe('all');
    expect(component.sort()).toBe('position');
  });
});
