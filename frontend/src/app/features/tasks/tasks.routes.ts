import { Routes } from '@angular/router';

export const TASK_ROUTES: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./pages/task-list-page.component').then((m) => m.TaskListPageComponent),
  },
  {
    path: 'board',
    loadComponent: () =>
      import('./pages/task-board-page.component').then((m) => m.TaskBoardPageComponent),
  },
  {
    path: 'new',
    loadComponent: () =>
      import('./pages/task-form-page.component').then((m) => m.TaskFormPageComponent),
  },
  {
    path: ':id/edit',
    loadComponent: () =>
      import('./pages/task-form-page.component').then((m) => m.TaskFormPageComponent),
  },
  {
    path: ':id',
    loadComponent: () =>
      import('./pages/task-detail-page.component').then((m) => m.TaskDetailPageComponent),
  },
];
