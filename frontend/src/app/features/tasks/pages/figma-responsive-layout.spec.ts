import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { DashboardPageComponent } from '../../dashboard/dashboard-page.component';
import { Task } from '../models/task.model';
import { TaskDataService } from '../services/task-data.service';
import { TaskFormPageComponent } from './task-form-page.component';
import { TaskListPageComponent } from './task-list-page.component';

const tasks: Task[] = [
  {
    id: 1,
    title: 'Preparar release',
    description: 'Revisar checklist',
    status: 'doing',
    priority: 'medium',
    due_date: null,
    position: 0,
    created_at: '',
    updated_at: '',
  },
];

describe('Figma responsive layouts', () => {
  const data = {
    list: vi.fn(() => of(tasks)),
    get: vi.fn(() => of(tasks[0])),
    create: vi.fn(() => of(tasks[0])),
    update: vi.fn(() => of(tasks[0])),
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: TaskDataService, useValue: data }],
    });
  });

  it('renders the dedicated mobile dashboard composition', () => {
    const fixture = TestBed.createComponent(DashboardPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.mobile-dashboard')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.mobile-progress')?.textContent).toContain('concluídas');
  });

  it('renders the compact mobile task list composition', () => {
    const fixture = TestBed.createComponent(TaskListPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.mobile-task-list')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.mobile-task-list .mobile-status')?.textContent).toContain('EM CURSO');
  });

  it('marks form actions for the single-action mobile composition', () => {
    const fixture = TestBed.createComponent(TaskFormPageComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.mobile-form-heading')?.textContent).toContain('Nova tarefa.');
    expect(fixture.nativeElement.querySelector('.desktop-cancel')).not.toBeNull();
  });
});
