export type TaskStatus = 'todo' | 'doing' | 'done';

export interface Task {
  id: number;
  user_id: number;
  title: string;
  description: string;
  status: TaskStatus;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface TaskPayload {
  title: string;
  description?: string;
  status?: TaskStatus;
}

export interface TaskMovePayload {
  status: TaskStatus;
  before_id: number | null;
  after_id: number | null;
}

export type TaskFeedback = 'saved' | 'failed';

export const STATUSES: TaskStatus[] = ['todo', 'doing', 'done'];

export interface StatusMeta {
  index: string;
  label: string;
  short: string;
}

export const STATUS_META: Record<TaskStatus, StatusMeta> = {
  todo: { index: '01', label: 'A fazer', short: 'A FAZER' },
  doing: { index: '02', label: 'Em curso', short: 'EM CURSO' },
  done: { index: '03', label: 'Feito', short: 'FEITO' },
};
