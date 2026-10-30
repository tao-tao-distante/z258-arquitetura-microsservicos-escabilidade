import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Task, TaskStatus } from '../models/task.model';
import { TaskCardComponent } from './task-card.component';

const task: Task = {
  id: 7,
  user_id: 1,
  title: 'Revisar release',
  description: '',
  status: 'doing',
  priority: 'medium',
  due_date: null,
  position: 0,
  created_at: '',
  updated_at: '',
};

function setup() {
  TestBed.configureTestingModule({ imports: [TaskCardComponent], providers: [provideRouter([])] });
  const fixture = TestBed.createComponent(TaskCardComponent);
  fixture.componentRef.setInput('task', task);
  fixture.detectChanges();
  return fixture;
}

describe('TaskCardComponent', () => {
  it('offers only the other columns in the move menu', () => {
    const fixture = setup();
    const options = Array.from(fixture.nativeElement.querySelectorAll('select option:not([disabled])')) as HTMLOptionElement[];

    expect(options.map((option) => option.value)).toEqual(['todo', 'done']);
  });

  it('emits the chosen column and resets the menu', () => {
    const fixture = setup();
    const moved: TaskStatus[] = [];
    fixture.componentInstance.move.subscribe((status) => moved.push(status));
    const select = fixture.nativeElement.querySelector('select') as HTMLSelectElement;

    select.value = 'done';
    select.dispatchEvent(new Event('change'));

    expect(moved).toEqual(['done']);
    expect(select.value).toBe('');
  });

  it.each([
    { feedback: 'saved' as const, className: 'saved' },
    { feedback: 'failed' as const, className: 'failed' },
  ])('marks the card as $className', ({ feedback, className }) => {
    const fixture = setup();

    fixture.componentRef.setInput('feedback', feedback);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.task').classList).toContain(className);
  });
});
