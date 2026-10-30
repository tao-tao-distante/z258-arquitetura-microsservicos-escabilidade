package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/domain"
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
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs("a@b.com", "hash").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
			AddRow(1, "a@b.com", "", "hash", now))

	repo := NewPostgresUserRepository(db)
	u, err := repo.Create(context.Background(), "a@b.com", "hash")
	require.NoError(t, err)
	require.Equal(t, int64(1), u.ID)
	require.Equal(t, "a@b.com", u.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
		WillReturnError(errors.New("boom"))
	repo := NewPostgresUserRepository(db)
	_, err := repo.Create(context.Background(), "a@b.com", "hash")
	require.Error(t, err)
}

func TestCreateMapsUniqueViolationToEmailTaken(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
		WillReturnError(&pq.Error{Code: "23505"})
	repo := NewPostgresUserRepository(db)
	_, err := repo.Create(context.Background(), "a@b.com", "hash")
	require.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestFindByEmail(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, name, password_hash, created_at FROM users WHERE email")).
		WithArgs("a@b.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at"}).
			AddRow(7, "a@b.com", "Ana", "hash", now))
	repo := NewPostgresUserRepository(db)
	u, err := repo.FindByEmail(context.Background(), "a@b.com")
	require.NoError(t, err)
	require.Equal(t, int64(7), u.ID)
	require.Equal(t, "Ana", u.Name)
}

func TestFindByEmailNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, name, password_hash, created_at FROM users WHERE email")).
		WithArgs("none@b.com").
		WillReturnError(sql.ErrNoRows)
	repo := NewPostgresUserRepository(db)
	_, err := repo.FindByEmail(context.Background(), "none@b.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFindByEmailGenericError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, email, name, password_hash, created_at FROM users WHERE email")).
		WithArgs("a@b.com").
		WillReturnError(errors.New("boom"))
	repo := NewPostgresUserRepository(db)
	_, err := repo.FindByEmail(context.Background(), "a@b.com")
	require.Error(t, err)
}

func TestMigrate(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS users")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("ALTER TABLE users ADD COLUMN IF NOT EXISTS name")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, Migrate(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMigrateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS users")).
		WillReturnError(errors.New("boom"))
	require.Error(t, Migrate(context.Background(), db))
}

func TestMigrateAddNameColumnError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS users")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("ALTER TABLE users ADD COLUMN IF NOT EXISTS name")).
		WillReturnError(errors.New("boom"))
	require.ErrorContains(t, Migrate(context.Background(), db), "name column")
}
