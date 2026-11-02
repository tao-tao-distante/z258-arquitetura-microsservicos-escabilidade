import { HttpErrorResponse, provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
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
    create: vi.fn((_: TaskPayload) => of({ id: saved.id })),
    update: vi.fn((_: number, __: TaskPayload) => of({ id: saved.id })),
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
    expect(navigate).toHaveBeenCalledWith(['/tasks/board']);
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

  it('creates once, redirects to the board, and lists without loading detail or history', () => {
    const navigate = vi.fn(() => {
      TestBed.inject(TaskDataService).list().subscribe();
      return Promise.resolve(true);
    });
    TestBed.configureTestingModule({
      imports: [TaskFormPageComponent],
      providers: [
        { provide: Router, useValue: { navigate } },
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({}) } } },
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    const requests = TestBed.inject(HttpTestingController);
    const form = TestBed.createComponent(TaskFormPageComponent).componentInstance;
    form.title = 'Nova tarefa';

    form.submit();
    const create = requests.expectOne('/api/tasks/tasks');
    expect(create.request.method).toBe('POST');
    create.flush({ id: 7 }, { status: 201, statusText: 'Created' });

    expect(navigate).toHaveBeenCalledOnce();
    expect(navigate).toHaveBeenCalledWith(['/tasks/board']);
    const pending = requests.match(() => true);
    expect(pending).toHaveLength(1);
    expect(pending[0].request.urlWithParams).toBe('/api/tasks/tasks?limit=100&offset=0');
    expect(pending.some((request) => /\/tasks\/7(?:\/history)?$/.test(request.request.url))).toBe(false);
    pending[0].flush([]);
    requests.verify();
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

  it('redirects an edited task to its detail', () => {
    const { component, navigate } = setup('7');

    component.submit();

    expect(navigate).toHaveBeenCalledWith(['/tasks', 7]);
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

  it('binds every field, clears the due date and submits through the DOM', async () => {
    const { fixture, data } = setup(null);
    await fixture.whenStable();
    const root = fixture.nativeElement as HTMLElement;
    const input = async (selector: string, value: string) => {
      const control = root.querySelector(selector) as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
      control.value = value;
      control.dispatchEvent(new Event(control instanceof HTMLSelectElement ? 'change' : 'input', { bubbles: true }));
      fixture.detectChanges();
      await fixture.whenStable();
    };

    fixture.componentInstance.error.set('Erro anterior');
    await input('#task-title', 'Via formulário');
    await input('#task-description', 'Descrição');
    await input('#task-status', 'doing');
    await input('#task-priority', 'urgent');
    await input('#task-due-date', shiftDays(3));
    fixture.detectChanges();
    (root.querySelector('.clear-due') as HTMLButtonElement).click();
    await input('#task-due-date', shiftDays(4));
    fixture.detectChanges();
    await fixture.whenStable();
    expect(fixture.componentInstance.title).toBe('Via formulário');
    expect(fixture.componentInstance.priority).toBe('urgent');
    (root.querySelector('button[type="submit"]') as HTMLButtonElement).click();

    expect(data.create).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Via formulário',
      description: 'Descrição',
      status: 'doing',
      priority: 'urgent',
      due_date: shiftDays(4),
    }));
  });

  it('reports a detail loading failure', () => {
    const data = {
      get: vi.fn(() => throwError(() => new HttpErrorResponse({ status: 500 }))),
      create: vi.fn(),
      update: vi.fn(),
    };
    TestBed.configureTestingModule({
      imports: [TaskFormPageComponent],
      providers: [
        provideRouter([]),
        { provide: TaskDataService, useValue: data },
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: convertToParamMap({ id: '7' }) } } },
      ],
    });

    const component = TestBed.createComponent(TaskFormPageComponent).componentInstance;
    component.ngOnInit();

    expect(component.error()).toBe('Não foi possível carregar a tarefa.');
  });
});
