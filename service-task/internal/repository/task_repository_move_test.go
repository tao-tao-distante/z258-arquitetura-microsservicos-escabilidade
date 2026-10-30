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

const (
	lockSQL     = "SELECT id FROM tasks WHERE id = $1 AND user_id = $2 FOR UPDATE"
	neighborSQL = "SELECT position FROM tasks WHERE id = $1 AND user_id = $2 AND status = $3 AND id <> $4 FOR UPDATE"
	topSQL      = "SELECT COALESCE(MIN(position) - 1024, 0) FROM tasks WHERE user_id = $1 AND status = $2 AND id <> $3"
	renumberSQL = "UPDATE tasks AS t SET position = s.rn * 1024"
	moveSQL     = "UPDATE tasks SET status = $3, position = $4, updated_at = NOW() WHERE id = $1 AND user_id = $2"
)

func int64Ptr(v int64) *int64 { return &v }

func expectLock(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta(lockSQL)).WithArgs(int64(5), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
}

func expectNeighbor(mock sqlmock.Sqlmock, id int64, position float64) {
	mock.ExpectQuery(regexp.QuoteMeta(neighborSQL)).WithArgs(id, int64(1), "doing", int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"position"}).AddRow(position))
}

func expectMoveUpdate(mock sqlmock.Sqlmock, position float64) {
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(moveSQL)).WithArgs(int64(5), int64(1), "doing", position).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at", "position"}).
			AddRow(5, 1, "t", "d", "doing", now, now, position))
}

func TestMoveComputesPosition(t *testing.T) {
	tests := []struct {
		name   string
		move   domain.Move
		expect func(sqlmock.Sqlmock)
		want   float64
	}{
		{
			name: "between neighbors",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2), AfterID: int64Ptr(3)},
			expect: func(mock sqlmock.Sqlmock) {
				expectNeighbor(mock, 2, 1024)
				expectNeighbor(mock, 3, 2048)
			},
			want: 1536,
		},
		{
			name: "bottom of column",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2)},
			expect: func(mock sqlmock.Sqlmock) {
				expectNeighbor(mock, 2, 2048)
			},
			want: 3072,
		},
		{
			name: "top of column",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", AfterID: int64Ptr(3)},
			expect: func(mock sqlmock.Sqlmock) {
				expectNeighbor(mock, 3, 1024)
			},
			want: 0,
		},
		{
			name: "without neighbors goes to top",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing"},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(topSQL)).WithArgs(int64(1), "doing", int64(5)).
					WillReturnRows(sqlmock.NewRows([]string{"position"}).AddRow(-2048.0))
			},
			want: -2048,
		},
		{
			name: "renumbers column when gap is exhausted",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2), AfterID: int64Ptr(3)},
			expect: func(mock sqlmock.Sqlmock) {
				expectNeighbor(mock, 2, 1)
				expectNeighbor(mock, 3, 1+1e-9)
				mock.ExpectExec(regexp.QuoteMeta(renumberSQL)).WithArgs(int64(1), "doing", int64(5)).
					WillReturnResult(sqlmock.NewResult(0, 2))
				expectNeighbor(mock, 2, 1024)
				expectNeighbor(mock, 3, 2048)
			},
			want: 1536,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectBegin()
			expectLock(mock)
			tt.expect(mock)
			expectMoveUpdate(mock, tt.want)
			mock.ExpectCommit()

			task, err := NewPostgresTaskRepository(db).Move(context.Background(), tt.move)

			require.NoError(t, err)
			require.Equal(t, "doing", task.Status)
			require.Equal(t, tt.want, task.Position)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMoveFailures(t *testing.T) {
	between := domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2), AfterID: int64Ptr(3)}
	boom := errors.New("boom")
	tests := []struct {
		name    string
		move    domain.Move
		expect  func(sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "begin error",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(boom)
			},
			wantErr: boom,
		},
		{
			name: "task not found",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(lockSQL)).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "lock error",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(lockSQL)).WillReturnError(boom)
				mock.ExpectRollback()
			},
			wantErr: boom,
		},
		{
			name: "neighbor outside target column",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				mock.ExpectQuery(regexp.QuoteMeta(neighborSQL)).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: domain.ErrConflict,
		},
		{
			name: "neighbor query error",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1024)
				mock.ExpectQuery(regexp.QuoteMeta(neighborSQL)).WillReturnError(boom)
				mock.ExpectRollback()
			},
			wantErr: boom,
		},
		{
			name: "neighbors out of order",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 2048)
				expectNeighbor(mock, 3, 1024)
				mock.ExpectRollback()
			},
			wantErr: domain.ErrConflict,
		},
		{
			name: "renumber error",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1)
				expectNeighbor(mock, 3, 1+1e-9)
				mock.ExpectExec(regexp.QuoteMeta(renumberSQL)).WillReturnError(boom)
				mock.ExpectRollback()
			},
			wantErr: boom,
		},
		{
			name: "gap still exhausted after renumber",
			move: between,
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1)
				expectNeighbor(mock, 3, 1+1e-9)
				mock.ExpectExec(regexp.QuoteMeta(renumberSQL)).WillReturnResult(sqlmock.NewResult(0, 2))
				expectNeighbor(mock, 2, 1)
				expectNeighbor(mock, 3, 1+1e-9)
				mock.ExpectRollback()
			},
			wantErr: domain.ErrConflict,
		},
		{
			name: "top query error",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing"},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				mock.ExpectQuery(regexp.QuoteMeta(topSQL)).WillReturnError(boom)
				mock.ExpectRollback()
			},
			wantErr: boom,
		},
		{
			name: "update not found",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2)},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1024)
				mock.ExpectQuery(regexp.QuoteMeta(moveSQL)).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "update error",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2)},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1024)
				mock.ExpectQuery(regexp.QuoteMeta(moveSQL)).WillReturnError(boom)
				mock.ExpectRollback()
			},
			wantErr: boom,
		},
		{
			name: "commit error",
			move: domain.Move{ID: 5, UserID: 1, Status: "doing", BeforeID: int64Ptr(2)},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				expectLock(mock)
				expectNeighbor(mock, 2, 1024)
				expectMoveUpdate(mock, 2048)
				mock.ExpectCommit().WillReturnError(boom)
			},
			wantErr: boom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			tt.expect(mock)

			_, err := NewPostgresTaskRepository(db).Move(context.Background(), tt.move)

			require.ErrorIs(t, err, tt.wantErr)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMigrateAddsBoardPosition(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`(?s)ADD COLUMN IF NOT EXISTS position DOUBLE PRECISION.*UPDATE tasks SET position = -id \* 1024 WHERE position IS NULL.*idx_tasks_user_status_position`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, Migrate(context.Background(), db))
}
