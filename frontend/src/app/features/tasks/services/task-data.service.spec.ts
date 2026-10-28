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

  it('accepts 204 when deleting a task', () => {
    const service = TestBed.inject(TaskDataService);
    const requests = TestBed.inject(HttpTestingController);
    let completed = false;

    service.remove(7).subscribe({ complete: () => (completed = true) });
    requests.expectOne('/api/tasks/tasks/7').flush(null, { status: 204, statusText: 'No Content' });

    expect(completed).toBe(true);
  });
});
