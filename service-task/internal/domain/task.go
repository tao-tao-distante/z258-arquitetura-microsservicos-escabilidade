package domain

import (
	"errors"
	"time"
)

// Erros de domínio retornados pelo service-task.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
)

// Prioridades aceitas para uma tarefa.
const (
	PriorityLow     = "low"
	PriorityMedium  = "medium"
	PriorityHigh    = "high"
	PriorityUrgent  = "urgent"
	DefaultPriority = PriorityMedium
)

// Task representa uma tarefa persistida.
type Task struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *Date
	Position    float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskInput reúne os campos editáveis de uma tarefa.
type TaskInput struct {
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *Date
}

type Move struct {
	ID       int64
	UserID   int64
	Status   string
	BeforeID *int64
	AfterID  *int64
}

// Ordenações aceitas na listagem de tarefas.
const (
	SortPosition  = "position"
	SortPriority  = "priority"
	SortCreatedAt = "created_at"
	SortDueDate   = "due_date"
	OrderAsc      = "asc"
	OrderDesc     = "desc"
)

// ListQuery descreve filtros, ordenação e paginação da listagem de tarefas.
type ListQuery struct {
	UserID   int64
	Status   string
	Priority string
	Overdue  bool
	Sort     string
	Order    string
	Limit    int
	Offset   int
}
