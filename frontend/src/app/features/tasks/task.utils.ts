import {
  FieldChange,
  HistoryAction,
  PRIORITY_META,
  STATUS_META,
  Task,
  TaskPriority,
  TaskSort,
  TaskStatus,
} from './models/task.model';

const DAY_MS = 24 * 60 * 60 * 1000;

const pad = (value: number) => value.toString().padStart(2, '0');

/** Data local do navegador no formato YYYY-MM-DD, o mesmo usado por `due_date`. */
export function todayISO(now: Date = new Date()): string {
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** Converte YYYY-MM-DD em DD/MM/AAAA sem passar por `Date`, evitando deslocamento de fuso. */
export function formatDate(iso: string): string {
  const [year, month, day] = iso.split('-');
  return `${day}/${month}/${year}`;
}

/** Formata um instante ISO como DD/MM/AAAA HH:MM no horário local. */
export function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function daysBetween(fromISO: string, toISO: string): number {
  return Math.round((Date.parse(toISO) - Date.parse(fromISO)) / DAY_MS);
}

export type DueTone = 'overdue' | 'soon' | 'normal';

export interface DueInfo {
  label: string;
  tone: DueTone;
}

/** Descreve o prazo da tarefa em relação a hoje; tarefas concluídas nunca aparecem como atrasadas. */
export function dueInfo(task: Pick<Task, 'due_date' | 'status'>, today: string = todayISO()): DueInfo | null {
  if (!task.due_date) {
    return null;
  }
  if (task.status === 'done') {
    return { label: `Prazo ${formatDate(task.due_date)}`, tone: 'normal' };
  }
  const days = daysBetween(today, task.due_date);
  if (days < 0) {
    const late = -days;
    return { label: `Atrasada há ${late} ${late === 1 ? 'dia' : 'dias'}`, tone: 'overdue' };
  }
  if (days === 0) {
    return { label: 'Vence hoje', tone: 'soon' };
  }
  if (days === 1) {
    return { label: 'Vence amanhã', tone: 'soon' };
  }
  if (days <= 3) {
    return { label: `Vence em ${days} dias`, tone: 'soon' };
  }
  return { label: `Vence em ${formatDate(task.due_date)}`, tone: 'normal' };
}

export function isOverdue(task: Pick<Task, 'due_date' | 'status'>, today: string = todayISO()): boolean {
  return dueInfo(task, today)?.tone === 'overdue';
}

/** Ordena uma cópia das tarefas com as mesmas regras padrão da API. */
export function sortTasks(tasks: Task[], sort: TaskSort): Task[] {
  const sorted = [...tasks];
  switch (sort) {
    case 'priority':
      return sorted.sort((a, b) => PRIORITY_META[b.priority].rank - PRIORITY_META[a.priority].rank || b.id - a.id);
    case 'created_at':
      return sorted.sort((a, b) => b.created_at.localeCompare(a.created_at) || b.id - a.id);
    case 'due_date':
      return sorted.sort((a, b) => compareDueDates(a.due_date, b.due_date) || b.id - a.id);
    default:
      return sorted;
  }
}

function compareDueDates(a: string | null, b: string | null): number {
  if (a === b) {
    return 0;
  }
  if (a === null) {
    return 1;
  }
  if (b === null) {
    return -1;
  }
  return a.localeCompare(b);
}

export const ACTION_LABELS: Record<HistoryAction, string> = {
  created: 'Criada',
  updated: 'Editada',
  moved: 'Movida',
};

const FIELD_LABELS: Record<FieldChange['field'], string> = {
  title: 'Título',
  description: 'Descrição',
  status: 'Status',
  priority: 'Prioridade',
  due_date: 'Prazo',
};

function fieldValue(field: FieldChange['field'], value: string | null): string {
  if (value === null || value === '') {
    return '—';
  }
  switch (field) {
    case 'status':
      return STATUS_META[value as TaskStatus]?.label ?? value;
    case 'priority':
      return PRIORITY_META[value as TaskPriority]?.label ?? value;
    case 'due_date':
      return formatDate(value);
    case 'title':
      return `“${value}”`;
    default:
      return value;
  }
}

/** Texto legível de uma mudança do histórico, ex.: "Prioridade: Média → Alta". */
export function describeChange(change: FieldChange, action: HistoryAction): string {
  const label = FIELD_LABELS[change.field] ?? change.field;
  if (change.field === 'description') {
    return 'Descrição alterada';
  }
  if (action === 'created') {
    return `${label}: ${fieldValue(change.field, change.to)}`;
  }
  return `${label}: ${fieldValue(change.field, change.from)} → ${fieldValue(change.field, change.to)}`;
}
