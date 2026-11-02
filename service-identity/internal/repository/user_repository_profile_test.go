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
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
)

var userRowColumns = []string{"id", "email", "name", "password_hash", "created_at"}

func TestFindByID(t *testing.T) {
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		wantErr error
		wantAny bool
	}{
		{name: "found", rows: sqlmock.NewRows(userRowColumns).AddRow(3, "a@b.com", "Ana", "hash", time.Now())},
		{name: "not found", err: sql.ErrNoRows, wantErr: domain.ErrNotFound},
		{name: "generic error", err: errors.New("boom"), wantAny: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			query := mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, name, password_hash, created_at FROM users WHERE id")).
				WithArgs(int64(3))
			if tt.rows != nil {
				query.WillReturnRows(tt.rows)
			} else {
				query.WillReturnError(tt.err)
			}
			u, err := NewPostgresUserRepository(db).FindByID(context.Background(), 3)
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantAny:
				require.Error(t, err)
				require.NotErrorIs(t, err, domain.ErrNotFound)
			default:
				require.NoError(t, err)
				require.Equal(t, int64(3), u.ID)
				require.Equal(t, "Ana", u.Name)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateName(t *testing.T) {
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		wantErr error
		wantAny bool
	}{
		{name: "updated", rows: sqlmock.NewRows(userRowColumns).AddRow(3, "a@b.com", "Bia", "hash", time.Now())},
		{name: "not found", err: sql.ErrNoRows, wantErr: domain.ErrNotFound},
		{name: "generic error", err: errors.New("boom"), wantAny: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			query := mock.ExpectQuery(regexp.QuoteMeta("UPDATE users SET name = $2 WHERE id = $1")).
				WithArgs(int64(3), "Bia")
			if tt.rows != nil {
				query.WillReturnRows(tt.rows)
			} else {
				query.WillReturnError(tt.err)
			}
			u, err := NewPostgresUserRepository(db).UpdateName(context.Background(), 3, "Bia")
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantAny:
				require.Error(t, err)
				require.NotErrorIs(t, err, domain.ErrNotFound)
			default:
				require.NoError(t, err)
				require.Equal(t, "Bia", u.Name)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	tests := []struct {
		name    string
		result  sql.Result
		err     error
		wantErr error
		wantAny bool
	}{
		{name: "updated", result: sqlmock.NewResult(0, 1)},
		{name: "not found", result: sqlmock.NewResult(0, 0), wantErr: domain.ErrNotFound},
		{name: "exec error", err: errors.New("boom"), wantAny: true},
		{name: "rows affected error", result: sqlmock.NewErrorResult(errors.New("boom")), wantAny: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMock(t)
			exec := mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET password_hash = $2 WHERE id = $1")).
				WithArgs(int64(3), "newhash")
			if tt.err != nil {
				exec.WillReturnError(tt.err)
			} else {
				exec.WillReturnResult(tt.result)
			}
			err := NewPostgresUserRepository(db).UpdatePassword(context.Background(), 3, "newhash")
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantAny:
				require.Error(t, err)
				require.NotErrorIs(t, err, domain.ErrNotFound)
			default:
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
