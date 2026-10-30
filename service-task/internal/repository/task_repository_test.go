package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/domain"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func TestCreate(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks (user_id, title, description, status, position)")).
		WithArgs(int64(1), "title", "desc", "todo").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "title", "desc", "todo", now, now, 0.0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).
		WithArgs(sqlmock.AnyArg(), int64(1), sqlmock.AnyArg(), "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	repo := NewPostgresTaskRepository(db)
	task, err := repo.Create(context.Background(), &domain.Task{UserID: 1, Title: "title", Description: "desc", Status: "todo"})
	require.NoError(t, err)
	require.Equal(t, int64(1), task.ID)
}

func TestCreateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).WillReturnError(errors.New("boom"))
	mock.ExpectRollback()
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Create(context.Background(), &domain.Task{})
	require.Error(t, err)
}

func TestCreateRollsBackWhenOutboxInsertFails(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "title", "desc", "todo", now, now, 0.0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).WillReturnError(errors.New("outbox down"))
	mock.ExpectRollback()
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Create(context.Background(), &domain.Task{UserID: 1, Title: "title", Description: "desc", Status: "todo"})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestList(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, title, description, status, created_at, updated_at, position FROM tasks WHERE user_id = $1 ORDER BY position ASC, id DESC")).
		WithArgs(int64(1), 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "t", "d", "todo", now, now, 0.0))
	repo := NewPostgresTaskRepository(db)
	tasks, err := repo.List(context.Background(), 1, 50, 0)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
}

func TestListError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(errors.New("boom"))
	repo := NewPostgresTaskRepository(db)
	_, err := repo.List(context.Background(), 1, 50, 0)
	require.Error(t, err)
}

func TestListScanError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	repo := NewPostgresTaskRepository(db)
	_, err := repo.List(context.Background(), 1, 50, 0)
	require.Error(t, err)
}

func TestGet(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, title, description, status, created_at, updated_at, position FROM tasks WHERE id")).
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "t", "d", "todo", now, now, 0.0))
	repo := NewPostgresTaskRepository(db)
	task, err := repo.Get(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), task.ID)
}

func TestGetNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(sql.ErrNoRows)
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Get(context.Background(), 1, 1)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).WillReturnError(errors.New("boom"))
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Get(context.Background(), 1, 1)
	require.Error(t, err)
}

func TestUpdate(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("position = CASE WHEN status <> $4")).
		WithArgs(int64(1), "t", "d", "done", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "t", "d", "done", now, now, 0.0))
	repo := NewPostgresTaskRepository(db)
	task, err := repo.Update(context.Background(), &domain.Task{ID: 1, UserID: 1, Title: "t", Description: "d", Status: "done"})
	require.NoError(t, err)
	require.Equal(t, "done", task.Status)
}

func TestUpdateNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE tasks")).WillReturnError(sql.ErrNoRows)
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Update(context.Background(), &domain.Task{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE tasks")).WillReturnError(errors.New("boom"))
	repo := NewPostgresTaskRepository(db)
	_, err := repo.Update(context.Background(), &domain.Task{})
	require.Error(t, err)
}

func TestDelete(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WithArgs(int64(1), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	repo := NewPostgresTaskRepository(db)
	require.NoError(t, repo.Delete(context.Background(), 1, 1))
}

func TestDeleteNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	repo := NewPostgresTaskRepository(db)
	require.ErrorIs(t, repo.Delete(context.Background(), 1, 1), domain.ErrNotFound)
}

func TestDeleteError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).WillReturnError(errors.New("boom"))
	repo := NewPostgresTaskRepository(db)
	require.Error(t, repo.Delete(context.Background(), 1, 1))
}

func TestMigrate(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).
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

func TestCreateBeginError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin down"))
	_, err := NewPostgresTaskRepository(db).Create(context.Background(), &domain.Task{})
	require.Error(t, err)
}

func TestCreateCommitError(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tasks")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(1, 1, "title", "desc", "todo", now, now, 0.0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO task_outbox")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit down"))
	mock.ExpectRollback()
	_, err := NewPostgresTaskRepository(db).Create(context.Background(), &domain.Task{UserID: 1, Title: "title", Status: "todo"})
	require.Error(t, err)
}

func TestListIterateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).RowError(0, errors.New("rows down")))
	_, err := NewPostgresTaskRepository(db).List(context.Background(), 1, 50, 0)
	require.Error(t, err)
}

func TestDeleteRowsAffectedError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tasks")).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected down")))
	require.Error(t, NewPostgresTaskRepository(db).Delete(context.Background(), 1, 1))
}
