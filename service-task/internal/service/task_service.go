package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
)

// TaskService implementa as regras de negócio de tarefas.
type TaskService struct {
	repo repository.TaskRepository
	now  func() time.Time
}

// New cria o serviço de tarefas com o repositório informado.
func New(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo, now: time.Now}
}

// Create valida os campos e cria a tarefa; prazo no passado é rejeitado.
func (s *TaskService) Create(ctx context.Context, userID int64, in domain.TaskInput) (*domain.Task, error) {
	task, err := normalizeInput(in)
	if err != nil {
		return nil, err
	}
	if task.DueDate != nil && s.isPast(*task.DueDate) {
		return nil, domain.ErrInvalidInput
	}
	task.UserID = userID
	created, err := s.repo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return created, nil
}

// List normaliza filtros e ordenação e devolve as tarefas do usuário.
func (s *TaskService) List(ctx context.Context, q domain.ListQuery) ([]domain.Task, error) {
	q, err := normalizeListQuery(q)
	if err != nil {
		return nil, err
	}
	tasks, err := s.repo.List(ctx, q)
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

// Update valida os campos e atualiza a tarefa do usuário. Um prazo no passado só é
// aceito quando não mudou, para permitir editar tarefas já atrasadas.
func (s *TaskService) Update(ctx context.Context, id, userID int64, in domain.TaskInput) (*domain.Task, error) {
	task, err := normalizeInput(in)
	if err != nil {
		return nil, err
	}
	if task.DueDate != nil && s.isPast(*task.DueDate) {
		current, err := s.repo.Get(ctx, id, userID)
		if err != nil {
			return nil, fmt.Errorf("update task: %w", err)
		}
		if !domain.SameDate(current.DueDate, task.DueDate) {
			return nil, domain.ErrInvalidInput
		}
	}
	task.ID, task.UserID = id, userID
	updated, err := s.repo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return updated, nil
}

// Delete remove uma tarefa do usuário.
func (s *TaskService) Delete(ctx context.Context, id, userID int64) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (s *TaskService) Move(ctx context.Context, move domain.Move) (*domain.Task, error) {
	move.Status = strings.TrimSpace(move.Status)
	if move.Status == "" || !validNeighbors(move) {
		return nil, domain.ErrInvalidInput
	}
	status, err := normalizeStatus(move.Status)
	if err != nil {
		return nil, err
	}
	move.Status = status
	task, err := s.repo.Move(ctx, move)
	if err != nil {
		return nil, fmt.Errorf("move task: %w", err)
	}
	return task, nil
}

// History devolve as alterações registradas da tarefa do usuário.
func (s *TaskService) History(ctx context.Context, id, userID int64, limit, offset int) ([]domain.HistoryEntry, error) {
	entries, err := s.repo.History(ctx, id, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("task history: %w", err)
	}
	return entries, nil
}

// isPast compara com o dia anterior em UTC: a API não conhece o fuso do cliente, e a
// folga de um dia evita rejeitar "hoje" para quem está atrás de UTC (ex.: UTC-3 à noite).
func (s *TaskService) isPast(d domain.Date) bool {
	floor := domain.NewDate(s.now()).AddDate(0, 0, -1)
	return d.Before(floor)
}

func validNeighbors(move domain.Move) bool {
	if move.BeforeID != nil && *move.BeforeID == move.ID {
		return false
	}
	if move.AfterID != nil && *move.AfterID == move.ID {
		return false
	}
	return move.BeforeID == nil || move.AfterID == nil || *move.BeforeID != *move.AfterID
}

func normalizeInput(in domain.TaskInput) (*domain.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, domain.ErrInvalidInput
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return nil, err
	}
	priority, err := normalizePriority(in.Priority)
	if err != nil {
		return nil, err
	}
	return &domain.Task{
		Title:       title,
		Description: in.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     in.DueDate,
	}, nil
}

func normalizeStatus(status string) (string, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return "todo", nil
	}
	switch status {
	case "todo", "doing", "done":
		return status, nil
	default:
		return "", domain.ErrInvalidInput
	}
}

func normalizePriority(priority string) (string, error) {
	priority = strings.TrimSpace(priority)
	if priority == "" {
		return domain.DefaultPriority, nil
	}
	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent:
		return priority, nil
	default:
		return "", domain.ErrInvalidInput
	}
}

// defaultOrder define a direção natural de cada ordenação: prioridade mais alta e
// tarefas mais novas primeiro; posição do quadro e prazo mais próximo em ordem crescente.
var defaultOrder = map[string]string{
	domain.SortPosition:  domain.OrderAsc,
	domain.SortPriority:  domain.OrderDesc,
	domain.SortCreatedAt: domain.OrderDesc,
	domain.SortDueDate:   domain.OrderAsc,
}

func normalizeListQuery(q domain.ListQuery) (domain.ListQuery, error) {
	var err error
	if q.Status = strings.TrimSpace(q.Status); q.Status != "" {
		if q.Status, err = normalizeStatus(q.Status); err != nil {
			return q, err
		}
	}
	if q.Priority = strings.TrimSpace(q.Priority); q.Priority != "" {
		if q.Priority, err = normalizePriority(q.Priority); err != nil {
			return q, err
		}
	}
	if q.Sort == "" {
		q.Sort = domain.SortPosition
	}
	order, ok := defaultOrder[q.Sort]
	if !ok {
		return q, domain.ErrInvalidInput
	}
	switch q.Order {
	case "":
		q.Order = order
	case domain.OrderAsc, domain.OrderDesc:
	default:
		return q, domain.ErrInvalidInput
	}
	return q, nil
}
