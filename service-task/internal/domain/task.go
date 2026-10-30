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

// Task representa uma tarefa persistida.
type Task struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Position    float64   `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Move struct {
	ID       int64
	UserID   int64
	Status   string
	BeforeID *int64
	AfterID  *int64
}
