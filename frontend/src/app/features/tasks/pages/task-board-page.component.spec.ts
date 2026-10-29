import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { TaskDataService } from '../services/task-data.service';
import { TaskBoardPageComponent } from './task-board-page.component';
import { Task } from '../models/task.model';

const base: Task = {
  id: 1,
  user_id: 1,
  title: 'Test',
  description: '',
  status: 'todo',
  created_at: '',
  updated_at: '',
};

describe('TaskBoardPageComponent', () => {
  it('groups loaded tasks by status', () => {
    const data = {
      list: vi.fn(() =>
        of([
          { ...base, id: 1, status: 'todo' as const },
          { ...base, id: 2, status: 'doing' as const },
          { ...base, id: 3, status: 'done' as const },
        ]),
      ),
    };
    TestBed.configureTestingModule({
      imports: [TaskBoardPageComponent],
      providers: [provideRouter([]), { provide: TaskDataService, useValue: data }],
    });
    const component = TestBed.createComponent(TaskBoardPageComponent).componentInstance;
    component.ngOnInit();

    expect(component.byStatus('todo').map((task) => task.id)).toEqual([1]);
    expect(component.byStatus('doing').map((task) => task.id)).toEqual([2]);
    expect(component.byStatus('done').map((task) => task.id)).toEqual([3]);
  });

  it('reports an oversized payload for a 413 response', () => {
    const data = {
      list: vi.fn(() =>
        throwError(() => new HttpErrorResponse({ status: 413, statusText: 'Payload Too Large' })),
      ),
    };
    TestBed.configureTestingModule({
      imports: [TaskBoardPageComponent],
      providers: [provideRouter([]), { provide: TaskDataService, useValue: data }],
    });
    const component = TestBed.createComponent(TaskBoardPageComponent).componentInstance;

    component.ngOnInit();

    expect(component.error()).toBe('O conteúdo enviado excede o limite permitido.');
  });
});
