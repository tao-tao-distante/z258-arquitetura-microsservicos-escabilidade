export type TaskStatus = 'todo' | 'doing' | 'done';
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent';

export interface Task {
  id: number;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  due_date: string | null;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface TaskMutationResponse {
  id: number;
}

export interface TaskPayload {
  title: string;
  description?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  due_date?: string | null;
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

export const PRIORITIES: TaskPriority[] = ['urgent', 'high', 'medium', 'low'];

export const DEFAULT_PRIORITY: TaskPriority = 'medium';

export interface PriorityMeta {
  label: string;
  short: string;
  rank: number;
}

export const PRIORITY_META: Record<TaskPriority, PriorityMeta> = {
  urgent: { label: 'Urgente', short: 'URGENTE', rank: 4 },
  high: { label: 'Alta', short: 'ALTA', rank: 3 },
  medium: { label: 'Média', short: 'MÉDIA', rank: 2 },
  low: { label: 'Baixa', short: 'BAIXA', rank: 1 },
};

export type TaskSort = 'position' | 'priority' | 'created_at' | 'due_date';

export const SORT_LABELS: Record<TaskSort, string> = {
  position: 'Ordem do quadro',
  priority: 'Prioridade',
  created_at: 'Mais recentes',
  due_date: 'Prazo mais próximo',
};

export type HistoryAction = 'created' | 'updated' | 'moved';
export type HistoryField = 'title' | 'description' | 'status' | 'priority' | 'due_date';

export interface FieldChange {
  field: HistoryField;
  from: string | null;
  to: string | null;
}

export interface TaskHistoryEntry {
  id: number;
  task_id: number;
  action: HistoryAction;
  changes: FieldChange[];
  changed_at: string;
}
