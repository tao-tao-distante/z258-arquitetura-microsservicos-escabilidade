package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) (*domain.Task, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error)
	Get(ctx context.Context, id, userID int64) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) (*domain.Task, error)
	Delete(ctx context.Context, id, userID int64) error
}

type PostgresTaskRepository struct {
	db *sql.DB
}

func NewPostgresTaskRepository(db *sql.DB) *PostgresTaskRepository {
	return &PostgresTaskRepository{db: db}
}

const columns = `id, user_id, title, description, status, created_at, updated_at`

func scanTask(row interface{ Scan(...any) error }) (*domain.Task, error) {
	var t domain.Task
	if err := row.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresTaskRepository) Create(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const query = `INSERT INTO tasks (user_id, title, description, status) VALUES ($1, $2, $3, $4)
		RETURNING ` + columns
	row := tx.QueryRowContext(ctx, query, task.UserID, task.Title, task.Description, task.Status)
	t, err := scanTask(row)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	eventID := uuid.NewString()
	payload, err := json.Marshal(map[string]any{"task_id": t.ID, "title": t.Title})
	if err != nil {
		return nil, fmt.Errorf("marshal task event: %w", err)
	}
	const outboxQuery = `INSERT INTO task_outbox (event_id, task_id, event_type, payload, correlation_id)
		VALUES ($1, $2, 'task.created', $3, $4)`
	if _, err := tx.ExecContext(ctx, outboxQuery, eventID, t.ID, payload, correlation.FromContext(ctx)); err != nil {
		return nil, fmt.Errorf("create outbox event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create task: %w", err)
	}
	return t, nil
}

func (r *PostgresTaskRepository) List(ctx context.Context, userID int64, limit, offset int) ([]domain.Task, error) {
	const query = `SELECT ` + columns + ` FROM tasks WHERE user_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *PostgresTaskRepository) Get(ctx context.Context, id, userID int64) (*domain.Task, error) {
	const query = `SELECT ` + columns + ` FROM tasks WHERE id = $1 AND user_id = $2`
	row := r.db.QueryRowContext(ctx, query, id, userID)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (r *PostgresTaskRepository) Update(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	const query = `UPDATE tasks SET title = $2, description = $3, status = $4, updated_at = NOW()
		WHERE id = $1 AND user_id = $5 RETURNING ` + columns
	row := r.db.QueryRowContext(ctx, query, task.ID, task.Title, task.Description, task.Status, task.UserID)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}

func (r *PostgresTaskRepository) Delete(ctx context.Context, id, userID int64) error {
	const query = `DELETE FROM tasks WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	const ddl = `CREATE TABLE IF NOT EXISTS tasks (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'todo',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_user_id_id_desc ON tasks (user_id, id DESC);
	CREATE TABLE IF NOT EXISTS task_outbox (
		id BIGSERIAL PRIMARY KEY,
		event_id UUID NOT NULL UNIQUE,
		task_id BIGINT NOT NULL,
		event_type TEXT NOT NULL,
		payload JSONB NOT NULL,
		correlation_id TEXT NOT NULL DEFAULT '',
		attempts INTEGER NOT NULL DEFAULT 0,
		available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		sent_at TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_task_outbox_pending ON task_outbox (available_at, id) WHERE sent_at IS NULL`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate tasks: %w", err)
	}
	return nil
}

type OutboxEvent struct {
	ID            int64
	EventID       string
	EventType     string
	Payload       []byte
	CorrelationID string
	Attempts      int
}

func (r *PostgresTaskRepository) ClaimOutbox(ctx context.Context) (*OutboxEvent, error) {
	const query = `WITH pending AS (
		SELECT id FROM task_outbox WHERE sent_at IS NULL AND available_at <= NOW()
		ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1
	) UPDATE task_outbox o SET attempts = attempts + 1, available_at = NOW() + INTERVAL '30 seconds'
	FROM pending WHERE o.id = pending.id
	RETURNING o.id, o.event_id, o.event_type, o.payload, o.correlation_id, o.attempts`
	var event OutboxEvent
	err := r.db.QueryRowContext(ctx, query).Scan(&event.ID, &event.EventID, &event.EventType, &event.Payload, &event.CorrelationID, &event.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim outbox event: %w", err)
	}
	return &event, nil
}

func (r *PostgresTaskRepository) MarkOutboxSent(ctx context.Context, id int64) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE task_outbox SET sent_at = NOW() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

func (r *PostgresTaskRepository) RetryOutbox(ctx context.Context, id int64, attempts int) error {
	delay := time.Second << min(attempts-1, 6)
	if _, err := r.db.ExecContext(ctx, `UPDATE task_outbox SET available_at = $2 WHERE id = $1`, id, time.Now().Add(delay)); err != nil {
		return fmt.Errorf("retry outbox event: %w", err)
	}
	return nil
}
