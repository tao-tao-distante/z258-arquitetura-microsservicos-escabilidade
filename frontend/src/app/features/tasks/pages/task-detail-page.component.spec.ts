import { HttpErrorResponse } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router, provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { Task } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { TaskDetailPageComponent } from './task-detail-page.component';

const task: Task = {
  id: 7,
  title: 'Release',
  description: 'Checklist',
  status: 'doing',
  priority: 'high',
  due_date: '2026-11-15',
  position: 1,
  created_at: '2026-10-30T10:00:00Z',
  updated_at: '2026-10-30T10:00:00Z',
};

describe('TaskDetailPageComponent', () => {
  let fixture: ComponentFixture<TaskDetailPageComponent>;
  const data = {
    get: vi.fn(() => of(task)),
    history: vi.fn(() => of([])),
    remove: vi.fn(() => of(undefined)),
  };

  async function setup(id: string | null): Promise<void> {
    await TestBed.configureTestingModule({
      imports: [TaskDetailPageComponent],
      providers: [
        provideRouter([]),
        { provide: TaskDataService, useValue: data },
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => id } } } },
      ],
    }).compileComponents();
    fixture = TestBed.createComponent(TaskDetailPageComponent);
    fixture.detectChanges();
  }

  beforeEach(() => {
    data.get.mockReset();
    data.get.mockReturnValue(of(task));
    data.history.mockReset();
    data.history.mockReturnValue(of([]));
    data.remove.mockReset();
    data.remove.mockReturnValue(of(undefined));
  });

  it('loads and presents task metadata', async () => {
    await setup('7');

    expect(data.get).toHaveBeenCalledWith(7);
    expect(fixture.componentInstance.meta('doing').label).toBe('Em curso');
    expect(fixture.componentInstance.taskUtils.formatDate(task.due_date!)).not.toBe('');
    expect(fixture.componentInstance.taskUtils.formatDateTime(task.created_at)).not.toBe('');
    expect(fixture.nativeElement.textContent).toContain('Checklist');
  });

  it('rejects an invalid route id without loading', async () => {
    await setup('invalid');

    expect(data.get).not.toHaveBeenCalled();
    expect(fixture.componentInstance.error()).toBe('Tarefa inválida.');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('reports a load failure', async () => {
    data.get.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 404 })));
    await setup('7');

    expect(fixture.componentInstance.task()).toBeNull();
    expect(fixture.componentInstance.error()).not.toBe('');
  });

  it('deletes the loaded task and returns to the list', async () => {
    await setup('7');
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate');

    fixture.componentInstance.remove();

    expect(data.remove).toHaveBeenCalledWith(7);
    expect(navigate).toHaveBeenCalledWith(['/tasks']);
    expect(fixture.componentInstance.pending()).toBe(false);
  });

  it('ignores deletion without a task or while another deletion is pending', async () => {
    await setup('invalid');
    fixture.componentInstance.remove();
    fixture.componentInstance.task.set(task);
    fixture.componentInstance.pending.set(true);
    fixture.componentInstance.remove();

    expect(data.remove).not.toHaveBeenCalled();
  });

  it('closes confirmation and reports deletion failures', async () => {
    data.remove.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 500 })));
    await setup('7');
    fixture.componentInstance.confirmOpen.set(true);

    fixture.componentInstance.remove();

    expect(fixture.componentInstance.confirmOpen()).toBe(false);
    expect(fixture.componentInstance.error()).toBe('Não foi possível excluir a tarefa.');
    expect(fixture.componentInstance.pending()).toBe(false);
  });
});
