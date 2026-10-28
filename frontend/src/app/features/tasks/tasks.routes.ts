import { Routes } from '@angular/router';

export const TASK_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/task-board-page.component').then((m) => m.TaskBoardPageComponent),
  },
];
