import { HttpErrorResponse } from '@angular/common/http';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';
import { Task } from '../tasks/models/task.model';
import { TaskDataService } from '../tasks/services/task-data.service';
import { DashboardPageComponent } from './dashboard-page.component';

const tasks: Task[] = [
  { id: 1, title: 'Doing', description: '', status: 'doing', priority: 'high', due_date: null, position: 1, created_at: '', updated_at: '' },
  { id: 2, title: 'Done', description: '', status: 'done', priority: 'low', due_date: null, position: 2, created_at: '', updated_at: '' },
  { id: 3, title: 'Todo', description: '', status: 'todo', priority: 'medium', due_date: null, position: 3, created_at: '', updated_at: '' },
];

describe('DashboardPageComponent', () => {
  let fixture: ComponentFixture<DashboardPageComponent>;
  const data = { list: vi.fn(() => of(tasks)) };

  beforeEach(async () => {
    data.list.mockReset();
    data.list.mockReturnValue(of(tasks));
    await TestBed.configureTestingModule({
      imports: [DashboardPageComponent],
      providers: [provideRouter([]), { provide: TaskDataService, useValue: data }],
    }).compileComponents();
  });

  it('summarizes progress and prioritizes ongoing tasks', () => {
    fixture = TestBed.createComponent(DashboardPageComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.progress()).toBe(33);
    expect(fixture.componentInstance.count('todo')).toBe(1);
    expect(fixture.componentInstance.ongoing().map((task) => task.id)).toEqual([1]);
  });

  it('uses todo tasks when nothing is ongoing and limits the list to three', () => {
    data.list.mockReturnValue(of(Array.from({ length: 4 }, (_, index) => ({ ...tasks[2], id: index + 1 }))));
    fixture = TestBed.createComponent(DashboardPageComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.progress()).toBe(0);
    expect(fixture.componentInstance.ongoing()).toHaveLength(3);
  });

  it('renders the empty state with zero progress', () => {
    data.list.mockReturnValue(of([]));
    fixture = TestBed.createComponent(DashboardPageComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.progress()).toBe(0);
    expect(fixture.nativeElement.textContent).toContain('NADA EM CURSO');
  });

  it('reports list failures', () => {
    data.list.mockReturnValue(throwError(() => new HttpErrorResponse({ status: 500 })));
    fixture = TestBed.createComponent(DashboardPageComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.error()).toBe('Não foi possível carregar a visão geral.');
  });
});
