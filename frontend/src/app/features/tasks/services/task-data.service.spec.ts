import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { TaskDataService } from './task-data.service';

describe('TaskDataService', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
  });

  it('rejects an unexpected create status', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let failed = false;

    service.create({ title: 'Task' }).subscribe({ error: () => (failed = true) });
    requests.expectOne('/api/tasks/tasks').flush({ id: 1 }, { status: 200, statusText: 'OK' });

    expect(failed).toBe(true);
  });

  it('fetches every page of tasks', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let taskCount = 0;
    const task = (id: number) => ({
      id,
      user_id: 1,
      title: `Task ${id}`,
      description: '',
      status: 'todo' as const,
      created_at: '',
      updated_at: '',
    });

    service.list().subscribe((tasks) => (taskCount = tasks.length));

    requests
      .expectOne((request) => request.url === '/api/tasks/tasks' && request.params.get('offset') === '0')
      .flush(Array.from({ length: 100 }, (_, index) => task(index + 1)));
    requests
      .expectOne((request) => request.url === '/api/tasks/tasks' && request.params.get('offset') === '100')
      .flush([task(101)]);

    expect(taskCount).toBe(101);
    requests.verify();
  });

  it('fetches a single task by id', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let taskId = 0;

    service.get(7).subscribe((task) => (taskId = task.id));
    requests.expectOne('/api/tasks/tasks/7').flush({
      id: 7,
      user_id: 1,
      title: 'Task',
      description: '',
      status: 'todo',
      created_at: '',
      updated_at: '',
    });

    expect(taskId).toBe(7);
  });

  it('accepts 204 when deleting a task', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let completed = false;

    service.remove(7).subscribe({ complete: () => (completed = true) });
    requests.expectOne('/api/tasks/tasks/7').flush(null, { status: 204, statusText: 'No Content' });

    expect(completed).toBe(true);
  });
});
