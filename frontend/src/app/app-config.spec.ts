import { appConfig } from './app.config';
import { routes } from './app.routes';
import { TASK_ROUTES } from './features/tasks/tasks.routes';

describe('application routing configuration', () => {
  it('registers application providers and every public route', () => {
    expect(appConfig.providers).toHaveLength(3);
    expect(routes.map((route) => route.path)).toEqual([
      '',
      'login',
      'register',
      'dashboard',
      'tasks',
      'activities',
      'profile',
      'notifications',
      '**',
    ]);
    expect(TASK_ROUTES.map((route) => route.path)).toEqual(['', 'board', 'new', ':id/edit', ':id']);
  });
});
