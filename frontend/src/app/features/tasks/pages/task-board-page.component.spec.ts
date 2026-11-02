import { CdkDragDrop } from '@angular/cdk/drag-drop';
import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Observable, of, Subject, throwError } from 'rxjs';
import type { Mock } from 'vitest';
import { TaskDataService } from '../services/task-data.service';
import { TaskBoardPageComponent } from './task-board-page.component';
import { Task, TaskMovePayload, TaskStatus } from '../models/task.model';

const base: Task = {
  id: 1,
  title: 'Test',
  description: '',
  status: 'todo',
  priority: 'medium',
  due_date: null,
  position: 0,
  created_at: '',
  updated_at: '',
};

function task(id: number, status: TaskStatus, position: number): Task {
  return { ...base, id, status, position, title: `Task ${id}` };
}

interface DataStub {
  list: Mock<() => Observable<Task[]>>;
  move: Mock<(id: number, payload: TaskMovePayload) => Observable<Task>>;
}

function setup(tasks: Task[], move: DataStub['move'] = vi.fn(() => new Subject<Task>())) {
  const data: DataStub = { list: vi.fn(() => of(tasks.map((item) => ({ ...item })))), move };
  TestBed.configureTestingModule({
    imports: [TaskBoardPageComponent],
    providers: [provideRouter([]), { provide: TaskDataService, useValue: data }],
  });
  const fixture = TestBed.createComponent(TaskBoardPageComponent);
  const component = fixture.componentInstance;
  component.ngOnInit();
  return { fixture, component, data };
}

function dropEvent(
  item: Task,
  from: TaskStatus,
  to: TaskStatus,
  previousIndex: number,
  currentIndex: number,
): CdkDragDrop<TaskStatus, TaskStatus, Task> {
  return {
    item: { data: item },
    previousContainer: { data: from },
    container: { data: to },
    previousIndex,
    currentIndex,
  } as unknown as CdkDragDrop<TaskStatus, TaskStatus, Task>;
}

function ids(tasks: Task[]): number[] {
  return tasks.map((item) => item.id);
}

describe('TaskBoardPageComponent', () => {
  afterEach(() => vi.useRealTimers());

  it('groups loaded tasks by status ordered by position', () => {
    const { component } = setup([
      task(1, 'todo', 2048),
      task(2, 'doing', 0),
      task(3, 'done', 0),
      task(4, 'todo', -1024),
    ]);

    expect(ids(component.byStatus('todo'))).toEqual([4, 1]);
    expect(ids(component.byStatus('doing'))).toEqual([2]);
    expect(ids(component.byStatus('done'))).toEqual([3]);
  });

  it('renders a drop list per column and a draggable card per task', () => {
    const { fixture } = setup([task(1, 'todo', 0), task(2, 'doing', 0)]);

    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.cdk-drop-list')).toHaveLength(3);
    expect(fixture.nativeElement.querySelectorAll('app-task-card.cdk-drag')).toHaveLength(2);
    expect(fixture.nativeElement.textContent).not.toContain('PUT /tasks');
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

  it('moves a card to another column right away and confirms with the server', () => {
    vi.useFakeTimers();
    const response = new Subject<Task>();
    const { component, data } = setup(
      [task(1, 'todo', 0), task(2, 'doing', 0), task(3, 'doing', 1024)],
      vi.fn(() => response),
    );

    component.drop(dropEvent(task(1, 'todo', 0), 'todo', 'doing', 0, 1));

    expect(ids(component.byStatus('todo'))).toEqual([]);
    expect(ids(component.byStatus('doing'))).toEqual([2, 1, 3]);
    expect(component.isPending(1)).toBe(true);
    expect(data.move).toHaveBeenCalledWith(1, { status: 'doing', before_id: 2, after_id: 3 });

    response.next(task(1, 'doing', 512));
    response.complete();

    expect(component.byStatus('doing').find((item) => item.id === 1)?.position).toBe(512);
    expect(component.isPending(1)).toBe(false);
    expect(component.feedback()[1]).toBe('saved');

    vi.runAllTimers();
    expect(component.feedback()[1]).toBeUndefined();
  });

  it.each([
    { name: 'down to the bottom', from: 0, to: 2, order: [2, 3, 1], before: 3, after: null },
    { name: 'up to the top', from: 2, to: 0, order: [3, 1, 2], before: null, after: 1 },
    { name: 'into the middle', from: 0, to: 1, order: [2, 1, 3], before: 2, after: 3 },
  ])('reorders a card $name of its column', ({ from, to, order, before, after }) => {
    const tasks = [task(1, 'todo', 0), task(2, 'todo', 1024), task(3, 'todo', 2048)];
    const { component, data } = setup(tasks);

    component.drop(dropEvent(tasks[from], 'todo', 'todo', from, to));

    expect(ids(component.byStatus('todo'))).toEqual(order);
    expect(data.move).toHaveBeenCalledWith(tasks[from].id, { status: 'todo', before_id: before, after_id: after });
  });

  it('ignores a card dropped back where it was', () => {
    const { component, data } = setup([task(1, 'todo', 0)]);

    component.drop(dropEvent(task(1, 'todo', 0), 'todo', 'todo', 0, 0));

    expect(data.move).not.toHaveBeenCalled();
  });

  it('returns the card to its origin and shakes it when the move fails', () => {
    const { component, data } = setup(
      [task(1, 'todo', 0), task(2, 'todo', 1024), task(3, 'done', 0)],
      vi.fn(() => throwError(() => new HttpErrorResponse({ status: 500 }))),
    );

    component.drop(dropEvent(task(1, 'todo', 0), 'todo', 'done', 0, 1));

    expect(ids(component.byStatus('todo'))).toEqual([1, 2]);
    expect(ids(component.byStatus('done'))).toEqual([3]);
    expect(component.isPending(1)).toBe(false);
    expect(component.feedback()[1]).toBe('failed');
    expect(component.error()).toBe('Não foi possível mover a tarefa.');
    expect(data.list).toHaveBeenCalledTimes(1);
  });

  it('reloads the board when the server reports it is out of date', () => {
    const { component, data } = setup(
      [task(1, 'todo', 0), task(2, 'doing', 0)],
      vi.fn(() => throwError(() => new HttpErrorResponse({ status: 409 }))),
    );
    data.list.mockReturnValueOnce(of([task(1, 'doing', -1024), task(2, 'doing', 0)]));

    component.drop(dropEvent(task(1, 'todo', 0), 'todo', 'doing', 0, 0));

    expect(data.list).toHaveBeenCalledTimes(2);
    expect(ids(component.byStatus('doing'))).toEqual([1, 2]);
    expect(component.error()).toBe('O quadro mudou em outro lugar. Recarregamos as tarefas.');
  });

  it('moves a card to the top of a column from the card menu', () => {
    const { component, data } = setup(
      [task(1, 'todo', 0), task(2, 'done', 0), task(3, 'done', 1024)],
      vi.fn(() => new Subject<Task>()),
    );

    component.moveTo(component.byStatus('todo')[0], 'done');

    expect(ids(component.byStatus('done'))).toEqual([1, 2, 3]);
    expect(data.move).toHaveBeenCalledWith(1, { status: 'done', before_id: null, after_id: 2 });
  });

  it('places a card in an empty column from the card menu', () => {
    const { component, data } = setup([task(1, 'todo', 0)], vi.fn(() => new Subject<Task>()));

    component.moveTo(component.byStatus('todo')[0], 'doing');

    expect(ids(component.byStatus('doing'))).toEqual([1]);
    expect(data.move).toHaveBeenCalledWith(1, { status: 'doing', before_id: null, after_id: null });
  });

  it('ignores a menu move to the current column', () => {
    const { component, data } = setup([task(1, 'todo', 0)]);

    component.moveTo(component.byStatus('todo')[0], 'todo');

    expect(data.move).not.toHaveBeenCalled();
  });
});
