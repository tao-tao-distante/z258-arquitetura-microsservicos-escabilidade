import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, provideRouter, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { Task, TaskPayload } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { todayISO } from '../task.utils';
import { TaskFormPageComponent } from './task-form-page.component';

function shiftDays(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() + days);
  return todayISO(date);
}

const saved: Task = {
  id: 7,
  user_id: 1,
  title: 'Revisar release',
  description: 'Checklist',
  status: 'doing',
  priority: 'high',
  due_date: null,
  position: 0,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T10:00:00Z',
};

function setup(id: string | null, existing: Task = saved) {
  const data = {
    get: vi.fn(() => of(existing)),
    create: vi.fn((_: TaskPayload) => of(saved)),
    update: vi.fn((_: number, __: TaskPayload) => of(saved)),
  };
  TestBed.configureTestingModule({
    imports: [TaskFormPageComponent],
    providers: [
      provideRouter([]),
      { provide: TaskDataService, useValue: data },
      { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap(id === null ? {} : { id }) } } },
    ],
  });
  const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const fixture = TestBed.createComponent(TaskFormPageComponent);
  fixture.detectChanges();
  return { fixture, component: fixture.componentInstance, data, navigate };
}

describe('TaskFormPageComponent', () => {
  it('renders priority options from highest to lowest and a due date picker', () => {
    const { fixture } = setup(null);
    const element = fixture.nativeElement as HTMLElement;
    const options = Array.from(element.querySelectorAll('#task-priority option')).map((option) => option.textContent?.trim());

    expect(options).toEqual(['URGENTE', 'ALTA', 'MÉDIA', 'BAIXA']);
    expect(element.querySelector('#task-due-date')?.getAttribute('type')).toBe('date');
    expect(element.querySelector('#task-due-date')?.getAttribute('min')).toBe(todayISO());
  });

  it('creates with default priority and no due date', () => {
    const { component, data, navigate } = setup(null);
    component.title = 'Nova';

    component.submit();

    expect(data.create).toHaveBeenCalledWith({ title: 'Nova', description: '', status: 'todo', priority: 'medium', due_date: null });
    expect(navigate).toHaveBeenCalledWith(['/tasks', 7]);
  });

  it('creates with the chosen priority and due date', () => {
    const { component, data } = setup(null);
    const due = shiftDays(5);
    component.title = 'Nova';
    component.priority = 'urgent';
    component.dueDate = due;

    component.submit();

    expect(data.create).toHaveBeenCalledWith(expect.objectContaining({ priority: 'urgent', due_date: due }));
  });

  it('rejects a due date in the past', () => {
    const { fixture, component, data } = setup(null);
    component.title = 'Nova';
    component.dueDate = shiftDays(-1);

    component.submit();
    fixture.detectChanges();

    expect(data.create).not.toHaveBeenCalled();
    expect(component.error()).toBe('O prazo não pode estar no passado.');
    expect((fixture.nativeElement as HTMLElement).querySelector('#task-due-date-error')).not.toBeNull();
  });

  it('loads priority and due date when editing', () => {
    const due = shiftDays(10);
    const { component } = setup('7', { ...saved, priority: 'low', due_date: due });

    expect(component.priority).toBe('low');
    expect(component.dueDate).toBe(due);
  });

  it('keeps an overdue due date that was not changed', () => {
    const overdue = shiftDays(-3);
    const { component, data } = setup('7', { ...saved, due_date: overdue });

    component.submit();

    expect(component.dueDateInvalid()).toBe(false);
    expect(data.update).toHaveBeenCalledWith(7, expect.objectContaining({ due_date: overdue, priority: 'high' }));
  });

  it('does not allow moving an overdue due date to another past day', () => {
    const { component, data } = setup('7', { ...saved, due_date: shiftDays(-3) });
    component.dueDate = shiftDays(-2);

    component.submit();

    expect(data.update).not.toHaveBeenCalled();
  });

  it('clears the due date', () => {
    const { component, data } = setup('7', { ...saved, due_date: shiftDays(4) });

    component.clearDueDate();
    component.submit();

    expect(data.update).toHaveBeenCalledWith(7, expect.objectContaining({ due_date: null }));
  });

  it('explains API validation errors', () => {
    const { component, data } = setup(null);
    data.create.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 400 })));
    component.title = 'Nova';

    component.submit();

    expect(component.error()).toBe('Verifique o título, a prioridade e o prazo.');
  });
});
