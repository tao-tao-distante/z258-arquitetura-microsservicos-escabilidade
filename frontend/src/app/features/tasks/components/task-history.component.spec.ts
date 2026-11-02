import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { Observable, of, throwError } from 'rxjs';
import type { Mock } from 'vitest';
import { TaskHistoryEntry } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { TaskHistoryComponent } from './task-history.component';

type HistoryFn = (id: number, offset?: number, limit?: number) => Observable<TaskHistoryEntry[]>;

function entry(id: number, overrides: Partial<TaskHistoryEntry> = {}): TaskHistoryEntry {
  return {
    id,
    task_id: 7,
    action: 'updated',
    changes: [{ field: 'priority', from: 'medium', to: 'high' }],
    changed_at: new Date(2026, 9, 30, 14, 2).toISOString(),
    ...overrides,
  };
}

function setup(history: Mock<HistoryFn>) {
  TestBed.configureTestingModule({
    imports: [TaskHistoryComponent],
    providers: [{ provide: TaskDataService, useValue: { history } }],
  });
  const fixture = TestBed.createComponent(TaskHistoryComponent);
  fixture.componentRef.setInput('taskId', 7);
  fixture.detectChanges();
  return { fixture, element: fixture.nativeElement as HTMLElement };
}

describe('TaskHistoryComponent', () => {
  it('renders the timeline newest first with readable changes', () => {
    const history = vi.fn<HistoryFn>(() =>
      of([
        entry(3, { changes: [{ field: 'priority', from: 'medium', to: 'high' }, { field: 'description', from: null, to: null }] }),
        entry(2, { action: 'moved', changes: [{ field: 'status', from: 'todo', to: 'doing' }] }),
        entry(1, { action: 'created', changes: [{ field: 'status', from: null, to: 'todo' }] }),
      ]),
    );
    const { element } = setup(history);

    const items = Array.from(element.querySelectorAll('.timeline > li')) as HTMLElement[];
    expect(history).toHaveBeenCalledWith(7, 0, TaskDataService.historyPageSize);
    expect(items.map((item) => item.querySelector('strong')?.textContent)).toEqual(['Editada', 'Movida', 'Criada']);
    expect(items[0].querySelector('time')?.textContent).toBe('30/10/2026 14:02');
    expect(items[0].textContent).toContain('Prioridade: Média → Alta');
    expect(items[0].textContent).toContain('Descrição alterada');
    expect(items[1].textContent).toContain('Status: A fazer → Em curso');
    expect(items[2].textContent).toContain('Status: A fazer');
    expect(element.querySelector('.more')).toBeNull();
  });

  it('shows an empty state', () => {
    const { element } = setup(vi.fn<HistoryFn>(() => of([])));
    expect(element.textContent).toContain('Nenhuma alteração registrada.');
  });

  it('shows its own error without breaking the page', () => {
    const { element } = setup(vi.fn<HistoryFn>(() => throwError(() => new HttpErrorResponse({ status: 500 }))));

    expect(element.querySelector('[role="alert"]')?.textContent).toContain('Não foi possível carregar o histórico.');
    expect(element.textContent).not.toContain('Nenhuma alteração registrada.');
  });

  it('loads the next page on demand', () => {
    const pageSize = TaskDataService.historyPageSize;
    const firstPage = Array.from({ length: pageSize }, (_, index) => entry(100 - index));
    const history = vi
      .fn<HistoryFn>()
      .mockReturnValueOnce(of(firstPage))
      .mockReturnValueOnce(of([entry(1, { action: 'created' })]));
    const { fixture, element } = setup(history);

    (element.querySelector('.more') as HTMLButtonElement).click();
    fixture.detectChanges();

    expect(history).toHaveBeenLastCalledWith(7, pageSize, pageSize);
    expect(element.querySelectorAll('.timeline > li').length).toBe(pageSize + 1);
    expect(element.querySelector('.more')).toBeNull();
  });
});
