import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpParams, HttpResponse } from '@angular/common/http';
import { EMPTY, expand, map, Observable, reduce } from 'rxjs';
import { environment } from '../../../../environments/environment';
import { Task, TaskHistoryEntry, TaskMovePayload, TaskMutationResponse, TaskPayload } from '../models/task.model';

@Injectable({ providedIn: 'root' })
export class TaskDataService {
  private static readonly pageSize = 100;
  static readonly historyPageSize = 20;
  private readonly http = inject(HttpClient);
  private readonly api = environment.taskUrl;

  list(): Observable<Task[]> {
    return this.fetchPage(0).pipe(
      expand((page, index) =>
        page.length === TaskDataService.pageSize
          ? this.fetchPage((index + 1) * TaskDataService.pageSize)
          : EMPTY,
      ),
      reduce((tasks, page) => tasks.concat(page), [] as Task[]),
    );
  }

  get(id: number): Observable<Task> {
    return this.http
      .get<Task>(`${this.api}/tasks/${id}`, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  create(payload: TaskPayload): Observable<TaskMutationResponse> {
    return this.http
      .post<TaskMutationResponse>(`${this.api}/tasks`, payload, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 201)));
  }

  update(id: number, payload: TaskPayload): Observable<TaskMutationResponse> {
    return this.http
      .put<TaskMutationResponse>(`${this.api}/tasks/${id}`, payload, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  move(id: number, payload: TaskMovePayload): Observable<Task> {
    return this.http
      .patch<Task>(`${this.api}/tasks/${id}/move`, payload, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  history(id: number, offset = 0, limit = TaskDataService.historyPageSize): Observable<TaskHistoryEntry[]> {
    return this.http
      .get<TaskHistoryEntry[]>(`${this.api}/tasks/${id}/history`, {
        observe: 'response',
        params: new HttpParams().set('limit', limit).set('offset', offset),
      })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  remove(id: number): Observable<void> {
    return this.http
      .delete(`${this.api}/tasks/${id}`, { observe: 'response', responseType: 'text' })
      .pipe(map((response) => {
        this.assertStatus(response, 204);
      }));
  }

  private fetchPage(offset: number): Observable<Task[]> {
    return this.http
      .get<Task[]>(`${this.api}/tasks`, {
        observe: 'response',
        params: new HttpParams()
          .set('limit', TaskDataService.pageSize)
          .set('offset', offset),
      })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  private bodyForStatus<T>(response: HttpResponse<T>, expected: number): T {
    this.assertStatus(response, expected);
    if (response.body === null) {
      throw new HttpErrorResponse({ status: response.status, statusText: 'Empty response body', url: response.url ?? undefined });
    }
    return response.body;
  }

  private assertStatus(response: HttpResponse<unknown>, expected: number): void {
    if (response.status !== expected) {
      throw new HttpErrorResponse({
        status: response.status,
        statusText: `Expected HTTP ${expected}`,
        url: response.url ?? undefined,
      });
    }
  }
}
