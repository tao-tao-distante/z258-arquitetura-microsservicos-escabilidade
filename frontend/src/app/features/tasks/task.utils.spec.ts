import { FieldChange, HistoryAction, Task } from './models/task.model';
import { describeChange, dueInfo, formatDate, formatDateTime, isOverdue, sortTasks, todayISO } from './task.utils';

const base: Task = {
  id: 1,
  user_id: 1,
  title: 'Task',
  description: '',
  status: 'todo',
  priority: 'medium',
  due_date: null,
  position: 0,
  created_at: '2026-10-01T10:00:00Z',
  updated_at: '2026-10-01T10:00:00Z',
};

describe('task utils', () => {
  it('builds today from the local calendar date', () => {
    expect(todayISO(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05');
  });

  it('formats calendar dates without timezone shifts', () => {
    expect(formatDate('2026-11-01')).toBe('01/11/2026');
  });

  it('formats instants in local time', () => {
    const local = new Date(2026, 9, 30, 14, 2);
    expect(formatDateTime(local.toISOString())).toBe('30/10/2026 14:02');
    expect(formatDateTime('not a date')).toBe('—');
  });

  it.each([
    { due: null, status: 'todo', want: null },
    { due: '2026-10-20', status: 'done', want: { label: 'Prazo 20/10/2026', tone: 'normal' } },
    { due: '2026-10-29', status: 'todo', want: { label: 'Atrasada há 1 dia', tone: 'overdue' } },
    { due: '2026-10-20', status: 'doing', want: { label: 'Atrasada há 10 dias', tone: 'overdue' } },
    { due: '2026-10-30', status: 'todo', want: { label: 'Vence hoje', tone: 'soon' } },
    { due: '2026-10-31', status: 'todo', want: { label: 'Vence amanhã', tone: 'soon' } },
    { due: '2026-11-02', status: 'todo', want: { label: 'Vence em 3 dias', tone: 'soon' } },
    { due: '2026-11-15', status: 'todo', want: { label: 'Vence em 15/11/2026', tone: 'normal' } },
  ] as const)('describes due date $due ($status)', ({ due, status, want }) => {
    expect(dueInfo({ due_date: due, status }, '2026-10-30')).toEqual(want);
  });

  it('flags only open tasks past their due date as overdue', () => {
    expect(isOverdue({ due_date: '2026-10-01', status: 'todo' }, '2026-10-30')).toBe(true);
    expect(isOverdue({ due_date: '2026-10-01', status: 'done' }, '2026-10-30')).toBe(false);
    expect(isOverdue({ due_date: null, status: 'todo' }, '2026-10-30')).toBe(false);
  });

  describe('sortTasks', () => {
    const tasks: Task[] = [
      { ...base, id: 1, priority: 'low', due_date: null, created_at: '2026-10-03T00:00:00Z' },
      { ...base, id: 2, priority: 'urgent', due_date: '2026-11-20', created_at: '2026-10-01T00:00:00Z' },
      { ...base, id: 3, priority: 'high', due_date: '2026-11-05', created_at: '2026-10-02T00:00:00Z' },
      { ...base, id: 4, priority: 'high', due_date: null, created_at: '2026-10-04T00:00:00Z' },
    ];
    const ids = (list: Task[]) => list.map((task) => task.id);

    it.each([
      { sort: 'position', want: [1, 2, 3, 4] },
      { sort: 'priority', want: [2, 4, 3, 1] },
      { sort: 'created_at', want: [4, 1, 3, 2] },
      { sort: 'due_date', want: [3, 2, 4, 1] },
    ] as const)('sorts by $sort', ({ sort, want }) => {
      expect(ids(sortTasks(tasks, sort))).toEqual(want);
    });

    it('does not mutate the original list', () => {
      sortTasks(tasks, 'priority');
      expect(ids(tasks)).toEqual([1, 2, 3, 4]);
    });
  });

  it.each<{ change: FieldChange; action: HistoryAction; want: string }>([
    { change: { field: 'status', from: null, to: 'todo' }, action: 'created', want: 'Status: A fazer' },
    { change: { field: 'due_date', from: null, to: '2026-11-15' }, action: 'created', want: 'Prazo: 15/11/2026' },
    { change: { field: 'status', from: 'todo', to: 'doing' }, action: 'moved', want: 'Status: A fazer → Em curso' },
    { change: { field: 'priority', from: 'medium', to: 'high' }, action: 'updated', want: 'Prioridade: Média → Alta' },
    { change: { field: 'due_date', from: null, to: '2026-11-15' }, action: 'updated', want: 'Prazo: — → 15/11/2026' },
    { change: { field: 'due_date', from: '2026-11-15', to: null }, action: 'updated', want: 'Prazo: 15/11/2026 → —' },
    { change: { field: 'title', from: 'a', to: 'b' }, action: 'updated', want: 'Título: “a” → “b”' },
    { change: { field: 'description', from: null, to: null }, action: 'updated', want: 'Descrição alterada' },
    { change: { field: 'priority', from: 'medium', to: 'legacy' }, action: 'updated', want: 'Prioridade: Média → legacy' },
  ])('describes $change.field on $action as "$want"', ({ change, action, want }) => {
    expect(describeChange(change, action)).toBe(want);
  });
});
