package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TaskRepository define a persistência de tarefas, do histórico e do outbox.
type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) (*domain.Task, error)
	List(ctx context.Context, query domain.ListQuery) ([]domain.Task, error)
	Get(ctx context.Context, id, userID int64) (*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) (*domain.Task, error)
	Delete(ctx context.Context, id, userID int64) error
	Move(ctx context.Context, move domain.Move) (*domain.Task, error)
	History(ctx context.Context, taskID, userID int64, limit, offset int) ([]domain.HistoryEntry, error)
}

// PostgresTaskRepository implementa TaskRepository com PostgreSQL.
type PostgresTaskRepository struct {
	db *sql.DB
}

// NewPostgresTaskRepository cria o repositório com o pool informado.
func NewPostgresTaskRepository(db *sql.DB) *PostgresTaskRepository {
	return &PostgresTaskRepository{db: db}
}

const columns = `id, user_id, title, description, status, created_at, updated_at, position, priority, due_date`

const (
	positionStep = 1024
	minGap       = 1e-6
)

// priorityRank ordena prioridades da menor (1) para a maior (4).
const priorityRank = `CASE priority WHEN 'urgent' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 ELSE 1 END`

func scanTask(row interface{ Scan(...any) error }) (*domain.Task, error) {
	var t domain.Task
	var due sql.NullTime
	if err := row.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.Position, &t.Priority, &due); err != nil {
		return nil, err
	}
	if due.Valid {
		d := domain.NewDate(due.Time)
		t.DueDate = &d
	}
	return &t, nil
}

// dueDateArg converte o prazo opcional para o parâmetro SQL do tipo DATE.
func dueDateArg(d *domain.Date) any {
	if d == nil {
		return nil
	}
	return d.String()
}

// Create grava a tarefa, o histórico inicial e o evento task.created no outbox na mesma transação.
func (r *PostgresTaskRepository) Create(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const query = `INSERT INTO tasks (user_id, title, description, status, priority, due_date, position)
		VALUES ($1, $2, $3, $4, $5, $6,
			(SELECT COALESCE(MIN(position) - 1024, 0) FROM tasks WHERE user_id = $1 AND status = $4))
		RETURNING ` + columns
	row := tx.QueryRowContext(ctx, query, task.UserID, task.Title, task.Description, task.Status, task.Priority, dueDateArg(task.DueDate))
	t, err := scanTask(row)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	if err := insertHistory(ctx, tx, t, domain.ActionCreated, domain.InitialChanges(*t)); err != nil {
		return nil, err
	}
	eventID := uuid.NewString()
	payload, err := json.Marshal(map[string]any{"task_id": t.ID, "title": t.Title})
	if err != nil {
		return nil, fmt.Errorf("marshal task event: %w", err)
	}
	traceCarrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, traceCarrier)
	traceContext, err := json.Marshal(traceCarrier)
	if err != nil {
		return nil, fmt.Errorf("marshal trace context: %w", err)
	}
	const outboxQuery = `INSERT INTO task_outbox (event_id, task_id, event_type, payload, correlation_id, trace_context)
		VALUES ($1, $2, 'task.created', $3, $4, $5)`
	if _, err := tx.ExecContext(ctx, outboxQuery, eventID, t.ID, payload, correlation.FromContext(ctx), string(traceContext)); err != nil {
		return nil, fmt.Errorf("create outbox event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create task: %w", err)
	}
	return t, nil
}

// List devolve as tarefas do usuário com filtros, ordenação e paginação.
func (r *PostgresTaskRepository) List(ctx context.Context, q domain.ListQuery) ([]domain.Task, error) {
	where := []string{"user_id = $1"}
	args := []any{q.UserID}
	if q.Status != "" {
		args = append(args, q.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if q.Priority != "" {
		args = append(args, q.Priority)
		where = append(where, fmt.Sprintf("priority = $%d", len(args)))
	}
	if q.Overdue {
		where = append(where, "due_date < CURRENT_DATE AND status <> 'done'")
	}
	args = append(args, q.Limit, q.Offset)
	query := `SELECT ` + columns + ` FROM tasks WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY ` + orderBy(q) + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.QueryContext(ctx, query, args...)
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

// orderBy monta a ordenação a partir de valores fixos; nunca interpola entrada do usuário.
func orderBy(q domain.ListQuery) string {
	dir := "ASC"
	if q.Order == domain.OrderDesc {
		dir = "DESC"
	}
	switch q.Sort {
	case domain.SortPriority:
		return priorityRank + " " + dir + ", id DESC"
	case domain.SortCreatedAt:
		return "created_at " + dir + ", id DESC"
	case domain.SortDueDate:
		return "due_date " + dir + " NULLS LAST, id DESC"
	default:
		return "position " + dir + ", id DESC"
	}
}

// Get busca uma tarefa pelo ID e pelo usuário.
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

// Update atualiza a tarefa do usuário e registra no histórico os campos alterados.
func (r *PostgresTaskRepository) Update(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const lockQuery = `SELECT ` + columns + ` FROM tasks WHERE id = $1 AND user_id = $2 FOR UPDATE`
	before, err := scanTask(tx.QueryRowContext(ctx, lockQuery, task.ID, task.UserID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock task: %w", err)
	}
	const query = `UPDATE tasks SET title = $2, description = $3, status = $4, priority = $5, due_date = $6,
		position = CASE WHEN status <> $4
			THEN (SELECT COALESCE(MIN(position) - 1024, 0) FROM tasks WHERE user_id = $7 AND status = $4)
			ELSE position END,
		updated_at = NOW()
		WHERE id = $1 AND user_id = $7 RETURNING ` + columns
	row := tx.QueryRowContext(ctx, query, task.ID, task.Title, task.Description, task.Status, task.Priority, dueDateArg(task.DueDate), task.UserID)
	after, err := scanTask(row)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	if changes := domain.Diff(*before, *after); len(changes) > 0 {
		if err := insertHistory(ctx, tx, after, domain.ActionUpdated, changes); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update task: %w", err)
	}
	return after, nil
}

// Delete remove a tarefa do usuário e devolve ErrNotFound quando ausente.
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

// Move reposiciona a tarefa no quadro e registra no histórico apenas trocas de coluna.
func (r *PostgresTaskRepository) Move(ctx context.Context, move domain.Move) (*domain.Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin move task: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const lockQuery = `SELECT status FROM tasks WHERE id = $1 AND user_id = $2 FOR UPDATE`
	var previousStatus string
	err = tx.QueryRowContext(ctx, lockQuery, move.ID, move.UserID).Scan(&previousStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock task: %w", err)
	}
	position, err := targetPosition(ctx, tx, move)
	if err != nil {
		return nil, err
	}
	const query = `UPDATE tasks SET status = $3, position = $4, updated_at = NOW() WHERE id = $1 AND user_id = $2
		RETURNING ` + columns
	t, err := scanTask(tx.QueryRowContext(ctx, query, move.ID, move.UserID, move.Status, position))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("move task: %w", err)
	}
	if previousStatus != t.Status {
		changes := []domain.FieldChange{domain.StatusChange(previousStatus, t.Status)}
		if err := insertHistory(ctx, tx, t, domain.ActionMoved, changes); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit move task: %w", err)
	}
	return t, nil
}

func targetPosition(ctx context.Context, tx *sql.Tx, move domain.Move) (float64, error) {
	if move.BeforeID == nil && move.AfterID == nil {
		const query = `SELECT COALESCE(MIN(position) - 1024, 0) FROM tasks WHERE user_id = $1 AND status = $2 AND id <> $3`
		var position float64
		if err := tx.QueryRowContext(ctx, query, move.UserID, move.Status, move.ID).Scan(&position); err != nil {
			return 0, fmt.Errorf("top position: %w", err)
		}
		return position, nil
	}
	for attempt := 0; ; attempt++ {
		prev, err := neighborPosition(ctx, tx, move.BeforeID, move)
		if err != nil {
			return 0, err
		}
		next, err := neighborPosition(ctx, tx, move.AfterID, move)
		if err != nil {
			return 0, err
		}
		switch {
		case next == nil:
			return *prev + positionStep, nil
		case prev == nil:
			return *next - positionStep, nil
		case *prev >= *next:
			return 0, domain.ErrConflict
		case *next-*prev >= minGap:
			return (*prev + *next) / 2, nil
		case attempt > 0:
			return 0, domain.ErrConflict
		}
		if err := renumber(ctx, tx, move); err != nil {
			return 0, err
		}
	}
}

func neighborPosition(ctx context.Context, tx *sql.Tx, id *int64, move domain.Move) (*float64, error) {
	if id == nil {
		return nil, nil
	}
	const query = `SELECT position FROM tasks WHERE id = $1 AND user_id = $2 AND status = $3 AND id <> $4 FOR UPDATE`
	var position float64
	err := tx.QueryRowContext(ctx, query, *id, move.UserID, move.Status, move.ID).Scan(&position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("neighbor position: %w", err)
	}
	return &position, nil
}

func renumber(ctx context.Context, tx *sql.Tx, move domain.Move) error {
	const query = `UPDATE tasks AS t SET position = s.rn * 1024
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position ASC, id DESC) AS rn
			FROM tasks WHERE user_id = $1 AND status = $2 AND id <> $3) AS s
		WHERE t.id = s.id`
	if _, err := tx.ExecContext(ctx, query, move.UserID, move.Status, move.ID); err != nil {
		return fmt.Errorf("renumber column: %w", err)
	}
	return nil
}

// insertHistory grava uma entrada de histórico dentro da transação da alteração.
func insertHistory(ctx context.Context, tx *sql.Tx, task *domain.Task, action string, changes []domain.FieldChange) error {
	payload, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("marshal task history: %w", err)
	}
	const query = `INSERT INTO task_history (task_id, user_id, action, changes, correlation_id)
		VALUES ($1, $2, $3, $4, $5)`
	if _, err := tx.ExecContext(ctx, query, task.ID, task.UserID, action, string(payload), correlation.FromContext(ctx)); err != nil {
		return fmt.Errorf("insert task history: %w", err)
	}
	return nil
}

// History devolve o histórico da tarefa do usuário, do mais recente para o mais antigo.
func (r *PostgresTaskRepository) History(ctx context.Context, taskID, userID int64, limit, offset int) ([]domain.HistoryEntry, error) {
	const ownerQuery = `SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, ownerQuery, taskID, userID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check task owner: %w", err)
	}
	if !exists {
		return nil, domain.ErrNotFound
	}
	const query = `SELECT id, task_id, action, changes, correlation_id, changed_at FROM task_history
		WHERE task_id = $1 ORDER BY changed_at DESC, id DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	defer rows.Close()

	var entries []domain.HistoryEntry
	for rows.Next() {
		var e domain.HistoryEntry
		var changes []byte
		if err := rows.Scan(&e.ID, &e.TaskID, &e.Action, &changes, &e.CorrelationID, &e.ChangedAt); err != nil {
			return nil, fmt.Errorf("scan task history: %w", err)
		}
		if err := json.Unmarshal(changes, &e.Changes); err != nil {
			return nil, fmt.Errorf("decode task history changes: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history: %w", err)
	}
	return entries, nil
}

// Migrate cria as tabelas tasks, task_history e task_outbox quando não existem.
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
	ALTER TABLE tasks ADD COLUMN IF NOT EXISTS position DOUBLE PRECISION;
	UPDATE tasks SET position = -id * 1024 WHERE position IS NULL;
	ALTER TABLE tasks ALTER COLUMN position SET DEFAULT 0;
	ALTER TABLE tasks ALTER COLUMN position SET NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_tasks_user_status_position ON tasks (user_id, status, position);
	ALTER TABLE tasks ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium'
		CHECK (priority IN ('low', 'medium', 'high', 'urgent'));
	ALTER TABLE tasks ADD COLUMN IF NOT EXISTS due_date DATE;
	CREATE INDEX IF NOT EXISTS idx_tasks_user_priority ON tasks (user_id, priority);
	CREATE INDEX IF NOT EXISTS idx_tasks_user_due_date ON tasks (user_id, due_date);
	CREATE TABLE IF NOT EXISTS task_history (
		id BIGSERIAL PRIMARY KEY,
		task_id BIGINT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
		user_id BIGINT NOT NULL,
		action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'moved')),
		changes JSONB NOT NULL DEFAULT '[]',
		correlation_id TEXT NOT NULL DEFAULT '',
		changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_task_history_task ON task_history (task_id, changed_at DESC, id DESC);
	INSERT INTO task_history (task_id, user_id, action, changes, changed_at)
	SELECT t.id, t.user_id, 'created', jsonb_build_array(
			jsonb_build_object('field', 'status', 'from', NULL, 'to', t.status),
			jsonb_build_object('field', 'priority', 'from', NULL, 'to', t.priority)),
		t.created_at
	FROM tasks t
	WHERE NOT EXISTS (SELECT 1 FROM task_history h WHERE h.task_id = t.id);
	CREATE TABLE IF NOT EXISTS task_outbox (
		id BIGSERIAL PRIMARY KEY,
		event_id UUID NOT NULL UNIQUE,
		task_id BIGINT NOT NULL,
		event_type TEXT NOT NULL,
		payload JSONB NOT NULL,
		correlation_id TEXT NOT NULL DEFAULT '',
		trace_context JSONB NOT NULL DEFAULT '{}',
		attempts INTEGER NOT NULL DEFAULT 0,
		available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		sent_at TIMESTAMPTZ
	);
	ALTER TABLE task_outbox ADD COLUMN IF NOT EXISTS trace_context JSONB NOT NULL DEFAULT '{}';
	CREATE INDEX IF NOT EXISTS idx_task_outbox_pending ON task_outbox (available_at, id) WHERE sent_at IS NULL`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate tasks: %w", err)
	}
	return nil
}

// OutboxEvent representa um evento pendente de publicação.
type OutboxEvent struct {
	ID            int64
	EventID       string
	EventType     string
	Payload       []byte
	CorrelationID string
	TraceContext  map[string]string
	Attempts      int
}

// ClaimOutbox reivindica um evento pendente com lease de 30 segundos.
func (r *PostgresTaskRepository) ClaimOutbox(ctx context.Context) (*OutboxEvent, error) {
	const query = `WITH pending AS (
		SELECT id FROM task_outbox WHERE sent_at IS NULL AND available_at <= NOW()
		ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1
	) UPDATE task_outbox o SET attempts = attempts + 1, available_at = NOW() + INTERVAL '30 seconds'
	FROM pending WHERE o.id = pending.id
	RETURNING o.id, o.event_id, o.event_type, o.payload, o.correlation_id, o.trace_context, o.attempts`
	var event OutboxEvent
	var traceContext []byte
	err := r.db.QueryRowContext(ctx, query).Scan(&event.ID, &event.EventID, &event.EventType, &event.Payload, &event.CorrelationID, &traceContext, &event.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim outbox event: %w", err)
	}
	if err := json.Unmarshal(traceContext, &event.TraceContext); err != nil {
		return nil, fmt.Errorf("decode outbox trace context: %w", err)
	}
	return &event, nil
}

// MarkOutboxSent registra o evento como publicado.
func (r *PostgresTaskRepository) MarkOutboxSent(ctx context.Context, id int64) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE task_outbox SET sent_at = NOW() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	return nil
}

// RetryOutbox agenda nova tentativa com backoff exponencial limitado a 64 segundos.
func (r *PostgresTaskRepository) RetryOutbox(ctx context.Context, id int64, attempts int) error {
	delay := time.Second << min(attempts-1, 6)
	if _, err := r.db.ExecContext(ctx, `UPDATE task_outbox SET available_at = $2 WHERE id = $1`, id, time.Now().Add(delay)); err != nil {
		return fmt.Errorf("retry outbox event: %w", err)
	}
	return nil
}
