import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpResponse } from '@angular/common/http';
import { map, Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import { Task, TaskPayload } from '../models/task.model';

@Injectable({ providedIn: 'root' })
export class TaskDataService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.taskUrl;

  list(): Observable<Task[]> {
    return this.http
      .get<Task[]>(`${this.api}/tasks`, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  create(payload: TaskPayload): Observable<Task> {
    return this.http
      .post<Task>(`${this.api}/tasks`, payload, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 201)));
  }

  update(id: number, payload: TaskPayload): Observable<Task> {
    return this.http
      .put<Task>(`${this.api}/tasks/${id}`, payload, { observe: 'response' })
      .pipe(map((response) => this.bodyForStatus(response, 200)));
  }

  remove(id: number): Observable<void> {
    return this.http
      .delete(`${this.api}/tasks/${id}`, { observe: 'response', responseType: 'text' })
      .pipe(map((response) => {
        this.assertStatus(response, 204);
      }));
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
