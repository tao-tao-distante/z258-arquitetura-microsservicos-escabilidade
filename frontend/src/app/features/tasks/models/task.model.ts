export type TaskStatus = 'todo' | 'doing' | 'done';

export interface Task {
  id: number;
  user_id: number;
  title: string;
  description: string;
  status: TaskStatus;
  created_at: string;
  updated_at: string;
}

export interface TaskPayload {
  title: string;
  description?: string;
  status?: TaskStatus;
}

export const STATUSES: TaskStatus[] = ['todo', 'doing', 'done'];
