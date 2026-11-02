package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/config"
)

type fakeChannel struct {
	err error
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return f.err
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, f.err
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return f.err
}

func (f *fakeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return f.err
}
func (f *fakeChannel) Confirm(bool) error                                             { return f.err }
func (f *fakeChannel) NotifyPublish(ch chan amqp.Confirmation) chan amqp.Confirmation { return ch }
func (f *fakeChannel) NotifyReturn(ch chan amqp.Return) chan amqp.Return              { return ch }

func (f *fakeChannel) Close() error { return nil }

type fakeConn struct {
	ch     Channel
	err    error
	closed bool
}

func (f *fakeConn) Channel() (Channel, error) { return f.ch, f.err }
func (f *fakeConn) Close() error              { return nil }
func (f *fakeConn) IsClosed() bool            { return f.closed }

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func expectDB(mock sqlmock.Sqlmock) {
	mock.ExpectPing()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).WillReturnResult(sqlmock.NewResult(0, 0))
}

func okDeps(db *sql.DB, conn Conn) Deps {
	return Deps{
		OpenDB:     func(string) (*sql.DB, error) { return db, nil },
		DialRabbit: func(string) (Conn, error) { return conn, nil },
	}
}

func TestRunOpenDBError(t *testing.T) {
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return nil, errors.New("boom") }}
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunWaitForDBCanceled(t *testing.T) {
	db, _ := newMock(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return db, nil }}
	require.Error(t, Run(ctx, config.Config{}, slog.Default(), deps))
}

func TestRunMigrateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).WillReturnError(errors.New("boom"))
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return db, nil }}
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunDialRabbitError(t *testing.T) {
	db, mock := newMock(t)
	expectDB(mock)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	deps := Deps{
		OpenDB:     func(string) (*sql.DB, error) { return db, nil },
		DialRabbit: func(string) (Conn, error) { return nil, errors.New("boom") },
	}
	require.Error(t, Run(ctx, config.Config{}, slog.Default(), deps))
}

func TestRunChannelError(t *testing.T) {
	db, mock := newMock(t)
	expectDB(mock)
	deps := okDeps(db, &fakeConn{err: errors.New("boom")})
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunSetupError(t *testing.T) {
	db, mock := newMock(t)
	expectDB(mock)
	deps := okDeps(db, &fakeConn{ch: &fakeChannel{err: errors.New("boom")}})
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunSuccess(t *testing.T) {
	db, mock := newMock(t)
	expectDB(mock)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()
	deps := okDeps(db, &fakeConn{ch: &fakeChannel{}})
	require.NoError(t, Run(ctx, config.Config{Port: "0"}, slog.Default(), deps))
}

func TestRunSuccessWithServerError(t *testing.T) {
	db, mock := newMock(t)
	expectDB(mock)
	deps := okDeps(db, &fakeConn{ch: &fakeChannel{}})
	require.Error(t, Run(context.Background(), config.Config{Port: "invalid"}, slog.Default(), deps))
}

func TestWaitForDBRetriesThenSucceeds(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing().WillReturnError(errors.New("down"))
	mock.ExpectPing()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, WaitForDB(ctx, db))
}

func TestNewServerAndShutdown(t *testing.T) {
	srv := NewServer("0", http.NewServeMux())
	require.Equal(t, ":0", srv.Addr)
	require.NotZero(t, srv.ReadTimeout)
	require.NotZero(t, srv.WriteTimeout)
	require.NotZero(t, srv.IdleTimeout)
	require.NoError(t, Shutdown(srv))
}
