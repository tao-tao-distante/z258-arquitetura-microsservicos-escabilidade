import { TestBed } from '@angular/core/testing';
import { Subject, of } from 'rxjs';
import { TaskDataService } from '../services/task-data.service';
import { TaskBoardPageComponent } from './task-board-page.component';
import { Task } from '../models/task.model';

const task: Task = {
  id: 1,
  user_id: 1,
  title: 'Test',
  description: '',
  status: 'todo',
  created_at: '',
  updated_at: '',
};

describe('TaskBoardPageComponent', () => {
  it('keeps pending state per task and ignores duplicate actions', () => {
    const updateResult = new Subject<Task>();
    const data = {
      list: vi.fn(() => of([task])),
      create: vi.fn(),
      update: vi.fn(() => updateResult),
      remove: vi.fn(),
    };
    TestBed.configureTestingModule({
      imports: [TaskBoardPageComponent],
      providers: [{ provide: TaskDataService, useValue: data }],
    });
    const component = TestBed.createComponent(TaskBoardPageComponent).componentInstance;
    component.ngOnInit();

    component.advance(task);
    component.advance(task);

    expect(component.isPending(task.id)).toBe(true);
    expect(data.update).toHaveBeenCalledOnce();
    updateResult.next({ ...task, status: 'doing' });
    updateResult.complete();
    expect(component.isPending(task.id)).toBe(false);
  });
});
