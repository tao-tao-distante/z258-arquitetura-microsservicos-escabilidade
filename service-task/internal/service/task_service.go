package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
)

// TaskService implementa as regras de negócio de tarefas.
type TaskService struct {
	repo repository.TaskRepository
}

// New cria o serviço de tarefas com o repositório informado.
func New(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// Create valida o título e cria a tarefa com status todo.
func (s *TaskService) Create(ctx context.Context, userID int64, title, description string) (*domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrInvalidInput
	}
	task, err := s.repo.Create(ctx, &domain.Task{
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      "todo",
	})
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

// List devolve as tarefas do usuário com paginação.
func (s *TaskService) List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error) {
	tasks, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Get devolve uma tarefa do usuário.
func (s *TaskService) Get(ctx context.Context, id, userID int64) (*domain.Task, error) {
	task, err := s.repo.Get(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return task, nil
}

// Update valida título e status e atualiza a tarefa do usuário.
func (s *TaskService) Update(ctx context.Context, id, userID int64, title, description, status string) (*domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrInvalidInput
	}
	if status == "" {
		status = "todo"
	}
	if status != "todo" && status != "doing" && status != "done" {
		return nil, domain.ErrInvalidInput
	}
	task, err := s.repo.Update(ctx, &domain.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      status,
	})
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return task, nil
}

// Delete remove uma tarefa do usuário.
func (s *TaskService) Delete(ctx context.Context, id, userID int64) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}
