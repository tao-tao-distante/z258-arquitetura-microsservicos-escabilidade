package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
)

const historySQL = "INSERT INTO task_history (task_id, user_id, action, changes, correlation_id)"

var taskColumns = []string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position", "priority", "due_date"}

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// taskRows devolve uma linha de tarefa com prioridade e prazo opcionais (nil = sem prazo).
func taskRows(id int64, title, status, priority string, due any) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(taskColumns).AddRow(id, 1, title, "d", status, now, now, 0.0, priority, due)
}

// jsonArg compara o argumento SQL como JSON, ignorando formatação.
type jsonArg string

func (j jsonArg) Match(v driver.Value) bool {
	raw, ok := v.(string)
	if !ok {
		return false
	}
	var got, want any
	if json.Unmarshal([]byte(raw), &got) != nil || json.Unmarshal([]byte(j), &want) != nil {
		return false
	}
	return reflect.DeepEqual(got, want)
}

func dueDate(t *testing.T, raw string) *domain.Date {
	t.Helper()
	d, err := domain.ParseDate(raw)
	require.NoError(t, err)
	return &d
}

func TestCreate(t *testing.T) {
	db, mock := newMock(t)
	due := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks (user_id, title, description, status, priority, due_date, position)")).
		WithArgs(int64(1), "title", "desc", "todo", "high", "2026-11-15").
		WillReturnRows(taskRows(1, "title", "todo", "high", due))
	mock.ExpectExec(regexp.QuoteMeta(historySQL)).
		WithArgs(int64(1), int64(1), "created",
			jsonArg(`[{"field":"status","from":null,"to":"todo"},{"field":"priority","from":null,"to":"high"},{"field":"due_date","from":null,"to":"2026-11-15"}]`),
			"cid-create").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).
		WithArgs(sqlmock.AnyArg(), int64(1), sqlmock.AnyArg(), "cid-create").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	repo := NewPostgresTaskRepository(db)
	ctx := correlation.NewContext(context.Background(), "cid-create")

	task, err := repo.Create(ctx, &domain.Task{UserID: 1, Title: "title", Description: "desc", Status: "todo", Priority: "high", DueDate: dueDate(t, "2026-11-15")})

	require.NoError(t, err)
	require.Equal(t, int64(1), task.ID)
	require.Equal(t, "high", task.Priority)
	require.Equal(t, "2026-11-15", task.DueDate.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateWithoutDueDate(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).
		WithArgs(int64(1), "title", "", "todo", "medium", nil).
		WillReturnRows(taskRows(1, "title", "todo", "medium", nil))
	mock.ExpectExec(regexp.QuoteMeta(historySQL)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	task, err := NewPostgresTaskRepository(db).Create(context.Background(), &domain.Task{UserID: 1, Title: "title", Status: "todo", Priority: "medium"})

	require.NoError(t, err)
	require.Nil(t, task.DueDate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFailures(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name   string
		expect func(sqlmock.Sqlmock)
	}{
		{name: "begin", expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin().WillReturnError(boom)
		}},
		{name: "insert task", expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "insert history", expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).WillReturnRows(taskRows(1, "title", "todo", "medium", nil))
			mock.ExpectExec(regexp.QuoteMeta(historySQL)).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "insert outbox", expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).WillReturnRows(taskRows(1, "title", "todo", "medium", nil))
			mock.ExpectExec(regexp.QuoteMeta(historySQL)).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "commit", expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).WillReturnRows(taskRows(1, "title", "todo", "medium", nil))
			mock.ExpectExec(regexp.QuoteMeta(historySQL)).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit().WillReturnError(boom)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			tt.expect(mock)

			_, err := NewPostgresTaskRepository(db).Create(context.Background(), &domain.Task{UserID: 1, Title: "title", Status: "todo", Priority: "medium"})

			require.ErrorIs(t, err, boom)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestList(t *testing.T) {
	const selectSQL = "SELECT id, user_id, title, description, status, created_at, updated_at, position, priority, due_date FROM tasks WHERE "
	tests := []struct {
		name  string
		query domain.ListQuery
		sql   string
		args  []driver.Value
	}{
		{
			name:  "default order by board position",
			query: domain.ListQuery{UserID: 1, Limit: 50},
			sql:   "user_id = $1 ORDER BY position ASC, id DESC LIMIT $2 OFFSET $3",
			args:  []driver.Value{int64(1), 50, 0},
		},
		{
			name:  "filters by status, priority and overdue",
			query: domain.ListQuery{UserID: 1, Status: "doing", Priority: "high", Overdue: true, Limit: 20, Offset: 40},
			sql:   "user_id = $1 AND status = $2 AND priority = $3 AND due_date < CURRENT_DATE AND status <> 'done' ORDER BY position ASC, id DESC LIMIT $4 OFFSET $5",
			args:  []driver.Value{int64(1), "doing", "high", 20, 40},
		},
		{
			name:  "sorts by priority rank",
			query: domain.ListQuery{UserID: 1, Sort: domain.SortPriority, Order: domain.OrderDesc, Limit: 50},
			sql:   "user_id = $1 ORDER BY " + priorityRank + " DESC, id DESC LIMIT $2 OFFSET $3",
			args:  []driver.Value{int64(1), 50, 0},
		},
		{
			name:  "sorts by creation date",
			query: domain.ListQuery{UserID: 1, Sort: domain.SortCreatedAt, Order: domain.OrderDesc, Limit: 50},
			sql:   "user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3",
			args:  []driver.Value{int64(1), 50, 0},
		},
		{
			name:  "sorts by due date keeping tasks without due date last",
			query: domain.ListQuery{UserID: 1, Sort: domain.SortDueDate, Order: domain.OrderAsc, Limit: 50},
			sql:   "user_id = $1 ORDER BY due_date ASC NULLS LAST, id DESC LIMIT $2 OFFSET $3",
			args:  []driver.Value{int64(1), 50, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectQuery("^" + regexp.QuoteMeta(selectSQL+tt.sql) + "$").
				WithArgs(tt.args...).
				WillReturnRows(taskRows(1, "t", "todo", "medium", nil))

			tasks, err := NewPostgresTaskRepository(db).List(context.Background(), tt.query)

			require.NoError(t, err)
			require.Len(t, tasks, 1)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestListError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(errors.New("boom"))
	_, err := NewPostgresTaskRepository(db).List(context.Background(), domain.ListQuery{UserID: 1, Limit: 50})
	require.Error(t, err)
}

func TestListScanError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	_, err := NewPostgresTaskRepository(db).List(context.Background(), domain.ListQuery{UserID: 1, Limit: 50})
	require.Error(t, err)
}

func TestListIterateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).RowError(0, errors.New("rows down")))
	_, err := NewPostgresTaskRepository(db).List(context.Background(), domain.ListQuery{UserID: 1, Limit: 50})
	require.Error(t, err)
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, title, description, status, created_at, updated_at, position, priority, due_date FROM tasks WHERE id")).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(taskRows(1, "t", "todo", "urgent", time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)))
	task, err := NewPostgresTaskRepository(db).Get(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), task.ID)
	require.Equal(t, "urgent", task.Priority)
	require.Equal(t, "2026-12-01", task.DueDate.String())
}

func TestGetNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(sql.ErrNoRows)
	_, err := NewPostgresTaskRepository(db).Get(context.Background(), 1, 1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(errors.New("boom"))
	_, err := NewPostgresTaskRepository(db).Get(context.Background(), 1, 1)
	require.Error(t, err)
}

const (
	updateLockSQL = "SELECT id, user_id, title, description, status, created_at, updated_at, position, priority, due_date FROM tasks WHERE id = $1 AND user_id = $2 FOR UPDATE"
	updateSQL     = "UPDATE tasks SET title = $2, description = $3, status = $4, priority = $5, due_date = $6"
)

func TestUpdateRecordsChangedFields(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WithArgs(int64(1), int64(1)).
		WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
	mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).
		WithArgs(int64(1), "t", "d", "done", "high", "2026-11-15", int64(1)).
		WillReturnRows(taskRows(1, "t", "done", "high", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)))
	mock.ExpectExec(regexp.QuoteMeta(historySQL)).
		WithArgs(int64(1), int64(1), "updated",
			jsonArg(`[{"field":"status","from":"todo","to":"done"},{"field":"priority","from":"medium","to":"high"},{"field":"due_date","from":null,"to":"2026-11-15"}]`),
			"").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	task, err := NewPostgresTaskRepository(db).Update(context.Background(), &domain.Task{
		ID: 1, UserID: 1, Title: "t", Description: "d", Status: "done", Priority: "high", DueDate: dueDate(t, "2026-11-15"),
	})

	require.NoError(t, err)
	require.Equal(t, "done", task.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWithoutChangesSkipsHistory(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
	mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).
		WithArgs(int64(1), "t", "d", "todo", "medium", nil, int64(1)).
		WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
	mock.ExpectCommit()

	_, err := NewPostgresTaskRepository(db).Update(context.Background(), &domain.Task{ID: 1, UserID: 1, Title: "t", Description: "d", Status: "todo", Priority: "medium"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr error
	}{
		{name: "begin", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin().WillReturnError(boom)
		}},
		{name: "not found", wantErr: domain.ErrNotFound, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnError(sql.ErrNoRows)
			mock.ExpectRollback()
		}},
		{name: "lock", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "update", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
			mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "history", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
			mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).WillReturnRows(taskRows(1, "novo", "todo", "medium", nil))
			mock.ExpectExec(regexp.QuoteMeta(historySQL)).WillReturnError(boom)
			mock.ExpectRollback()
		}},
		{name: "commit", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(updateLockSQL)).WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
			mock.ExpectQuery(regexp.QuoteMeta(updateSQL)).WillReturnRows(taskRows(1, "t", "todo", "medium", nil))
			mock.ExpectCommit().WillReturnError(boom)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			tt.expect(mock)

			_, err := NewPostgresTaskRepository(db).Update(context.Background(), &domain.Task{ID: 1, UserID: 1, Title: "novo", Status: "todo", Priority: "medium"})

			require.ErrorIs(t, err, tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDelete(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WithArgs(int64(1), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewPostgresTaskRepository(db).Delete(context.Background(), 1, 1))
}

func TestDeleteNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, NewPostgresTaskRepository(db).Delete(context.Background(), 1, 1), domain.ErrNotFound)
}

func TestDeleteError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).WillReturnError(errors.New("boom"))
	require.Error(t, NewPostgresTaskRepository(db).Delete(context.Background(), 1, 1))
}

func TestDeleteRowsAffectedError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected down")))
	require.Error(t, NewPostgresTaskRepository(db).Delete(context.Background(), 1, 1))
}

const (
	ownerSQL       = "SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND user_id = $2)"
	historyListSQL = "SELECT id, task_id, action, changes, correlation_id, changed_at FROM task_history"
)

var historyColumns = []string{"id", "task_id", "action", "changes", "correlation_id", "changed_at"}

func TestHistory(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(ownerSQL)).WithArgs(int64(5), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(historyListSQL) + `\s+` + regexp.QuoteMeta("WHERE task_id = $1 ORDER BY changed_at DESC, id DESC LIMIT $2 OFFSET $3")).
		WithArgs(int64(5), 20, 0).
		WillReturnRows(sqlmock.NewRows(historyColumns).
			AddRow(2, 5, "moved", []byte(`[{"field":"status","from":"todo","to":"doing"}]`), "cid-2", now).
			AddRow(1, 5, "created", []byte(`[{"field":"status","from":null,"to":"todo"}]`), "cid-1", now.Add(-time.Hour)))

	entries, err := NewPostgresTaskRepository(db).History(context.Background(), 5, 1, 20, 0)

	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "moved", entries[0].Action)
	require.Equal(t, "cid-2", entries[0].CorrelationID)
	require.Equal(t, []domain.FieldChange{domain.StatusChange("todo", "doing")}, entries[0].Changes)
	require.Nil(t, entries[1].Changes[0].From)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHistoryFailures(t *testing.T) {
	boom := errors.New("boom")
	owned := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(regexp.QuoteMeta(ownerSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	}
	tests := []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr error
	}{
		{name: "task of another user", wantErr: domain.ErrNotFound, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(regexp.QuoteMeta(ownerSQL)).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
		}},
		{name: "owner check", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			mock.ExpectQuery(regexp.QuoteMeta(ownerSQL)).WillReturnError(boom)
		}},
		{name: "query", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			owned(mock)
			mock.ExpectQuery(regexp.QuoteMeta(historyListSQL)).WillReturnError(boom)
		}},
		{name: "scan", expect: func(mock sqlmock.Sqlmock) {
			owned(mock)
			mock.ExpectQuery(regexp.QuoteMeta(historyListSQL)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		}},
		{name: "invalid changes json", expect: func(mock sqlmock.Sqlmock) {
			owned(mock)
			mock.ExpectQuery(regexp.QuoteMeta(historyListSQL)).
				WillReturnRows(sqlmock.NewRows(historyColumns).AddRow(1, 5, "created", []byte(`{`), "", time.Now()))
		}},
		{name: "iterate", wantErr: boom, expect: func(mock sqlmock.Sqlmock) {
			owned(mock)
			mock.ExpectQuery(regexp.QuoteMeta(historyListSQL)).
				WillReturnRows(sqlmock.NewRows(historyColumns).AddRow(1, 5, "created", []byte(`[]`), "", time.Now()).RowError(0, boom))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			tt.expect(mock)

			_, err := NewPostgresTaskRepository(db).History(context.Background(), 5, 1, 20, 0)

			require.Error(t, err)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMigrate(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, Migrate(context.Background(), db))
}

func TestMigrateAddsPriorityDueDateAndHistory(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`(?s)ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium'.*` +
		`ADD COLUMN IF NOT EXISTS due_date DATE.*` +
		`CREATE TABLE IF NOT EXISTS task_history.*REFERENCES tasks \(id\) ON DELETE CASCADE.*` +
		`INSERT INTO task_history .*WHERE NOT EXISTS`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, Migrate(context.Background(), db))
}

func TestMigrateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).WillReturnError(errors.New("boom"))
	require.Error(t, Migrate(context.Background(), db))
}

func TestClaimOutbox(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("WITH pending AS")).WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "event_type", "payload", "correlation_id", "attempts"}).
			AddRow(3, "event-1", "task.created", []byte(`{"task_id":1}`), "cid", 2),
	)
	event, err := NewPostgresTaskRepository(db).ClaimOutbox(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(3), event.ID)
	require.Equal(t, "cid", event.CorrelationID)
}

func TestClaimOutboxEmpty(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("WITH pending AS")).WillReturnError(sql.ErrNoRows)
	event, err := NewPostgresTaskRepository(db).ClaimOutbox(context.Background())
	require.NoError(t, err)
	require.Nil(t, event)
}

func TestClaimOutboxError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("WITH pending AS")).WillReturnError(errors.New("boom"))
	_, err := NewPostgresTaskRepository(db).ClaimOutbox(context.Background())
	require.Error(t, err)
}

func TestMarkOutboxSent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE task_outbox SET sent_at")).WithArgs(int64(3)).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewPostgresTaskRepository(db).MarkOutboxSent(context.Background(), 3))
}

func TestMarkOutboxSentError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE task_outbox SET sent_at")).WillReturnError(errors.New("boom"))
	require.Error(t, NewPostgresTaskRepository(db).MarkOutboxSent(context.Background(), 3))
}

func TestRetryOutbox(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE task_outbox SET available_at")).WithArgs(int64(3), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewPostgresTaskRepository(db).RetryOutbox(context.Background(), 3, 2))
}

func TestRetryOutboxError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("UPDATE task_outbox SET available_at")).WillReturnError(errors.New("boom"))
	require.Error(t, NewPostgresTaskRepository(db).RetryOutbox(context.Background(), 3, 2))
}
