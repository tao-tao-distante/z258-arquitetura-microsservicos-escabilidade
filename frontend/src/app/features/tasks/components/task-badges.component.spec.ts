import { TestBed } from '@angular/core/testing';
import { DueDateComponent } from './due-date.component';
import { PriorityBadgeComponent } from './priority-badge.component';

describe('PriorityBadgeComponent', () => {
  it.each([
    { priority: 'urgent', text: 'URGENTE' },
    { priority: 'high', text: 'ALTA' },
    { priority: 'medium', text: 'MÉDIA' },
    { priority: 'low', text: 'BAIXA' },
  ] as const)('renders $priority', ({ priority, text }) => {
    const fixture = TestBed.createComponent(PriorityBadgeComponent);
    fixture.componentRef.setInput('priority', priority);
    fixture.detectChanges();

    const badge = fixture.nativeElement.querySelector('.priority') as HTMLElement;
    expect(badge.textContent?.trim()).toBe(text);
    expect(badge.classList).toContain(priority);
  });
});

describe('DueDateComponent', () => {
  function render(task: { due_date: string | null; status: 'todo' | 'doing' | 'done' }) {
    const fixture = TestBed.createComponent(DueDateComponent);
    fixture.componentRef.setInput('task', task);
    fixture.componentRef.setInput('today', '2026-10-30');
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  it('renders nothing without a due date', () => {
    expect(render({ due_date: null, status: 'todo' }).querySelector('.due')).toBeNull();
  });

  it('highlights overdue tasks', () => {
    const due = render({ due_date: '2026-10-28', status: 'doing' }).querySelector('.due') as HTMLElement;
    expect(due.classList).toContain('overdue');
    expect(due.textContent).toContain('Atrasada há 2 dias');
  });

  it('shows upcoming due dates', () => {
    const due = render({ due_date: '2026-10-31', status: 'todo' }).querySelector('.due') as HTMLElement;
    expect(due.classList).toContain('soon');
    expect(due.textContent).toContain('Vence amanhã');
  });
});
