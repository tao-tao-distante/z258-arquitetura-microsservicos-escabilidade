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
      position: id,
      created_at: '',
      updated_at: '',
      priority: 'medium',
      due_date: null,
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
      position: 0,
      created_at: '',
      updated_at: '',
      priority: 'medium',
      due_date: null,
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

  it('moves a task with PATCH to the move endpoint', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let position = 0;

    service.move(7, { status: 'doing', before_id: 2, after_id: null }).subscribe((task) => (position = task.position));
    const request = requests.expectOne('/api/tasks/tasks/7/move');
    request.flush({
      id: 7,
      user_id: 1,
      title: 'Task',
      description: '',
      status: 'doing',
      position: 3072,
      created_at: '',
      updated_at: '',
      priority: 'medium',
      due_date: null,
    });

    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual({ status: 'doing', before_id: 2, after_id: null });
    expect(position).toBe(3072);
  });

  it('rejects an unexpected move status', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let failed = false;

    service.move(7, { status: 'done', before_id: null, after_id: null }).subscribe({ error: () => (failed = true) });
    requests.expectOne('/api/tasks/tasks/7/move').flush({ id: 7 }, { status: 202, statusText: 'Accepted' });

    expect(failed).toBe(true);
  });

  it('fetches a page of the task history', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let actions: string[] = [];

    service.history(7, 20).subscribe((entries) => (actions = entries.map((entry) => entry.action)));
    const request = requests.expectOne((req) => req.url === '/api/tasks/tasks/7/history');
    request.flush([{ id: 2, task_id: 7, action: 'moved', changes: [], changed_at: '2026-10-30T14:02:00Z' }]);

    expect(request.request.method).toBe('GET');
    expect(request.request.params.get('limit')).toBe('20');
    expect(request.request.params.get('offset')).toBe('20');
    expect(actions).toEqual(['moved']);
  });

  it('sends priority and due date when creating a task', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);

    service.create({ title: 'Task', priority: 'urgent', due_date: '2026-11-15' }).subscribe();
    const request = requests.expectOne('/api/tasks/tasks');
    request.flush({ id: 1 }, { status: 201, statusText: 'Created' });

    expect(request.request.body).toEqual({ title: 'Task', priority: 'urgent', due_date: '2026-11-15' });
  });
});
