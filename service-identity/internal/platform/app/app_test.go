package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/platform/config"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func expectMigration(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS users")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("ALTER TABLE users ADD COLUMN IF NOT EXISTS name")).WillReturnResult(sqlmock.NewResult(0, 0))
}

func TestRunOpenDBError(t *testing.T) {
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return nil, errors.New("boom") }}
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunWaitForDBContextCanceled(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectClose()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return db, nil }}
	require.Error(t, Run(ctx, config.Config{}, slog.Default(), deps))
}

func TestRunMigrateError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS users")).WillReturnError(errors.New("boom"))
	mock.ExpectClose()
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return db, nil }}
	require.Error(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunSuccess(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()
	deps := Deps{OpenDB: func(string) (*sql.DB, error) { return db, nil }}
	require.NoError(t, Run(ctx, config.Config{Port: "0"}, slog.Default(), deps))
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

func TestConfigureDBPool(t *testing.T) {
	db, _ := newMock(t)
	ConfigureDBPool(db)
	stats := db.Stats()
	require.Equal(t, 10, stats.MaxOpenConnections)
}

func TestRunPropagatesServeError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose()
	deps := Deps{
		OpenDB: func(string) (*sql.DB, error) { return db, nil },
		Serve:  func(*http.Server) error { return errors.New("listen failed") },
	}
	err := Run(context.Background(), config.Config{}, slog.Default(), deps)
	require.ErrorContains(t, err, "listen failed")
}

func TestRunAcceptsServerClosed(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose()
	deps := Deps{
		OpenDB: func(string) (*sql.DB, error) { return db, nil },
		Serve:  func(*http.Server) error { return http.ErrServerClosed },
	}
	require.NoError(t, Run(context.Background(), config.Config{}, slog.Default(), deps))
}

func TestRunReturnsDatabaseCloseError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose().WillReturnError(errors.New("close failed"))
	deps := Deps{
		OpenDB: func(string) (*sql.DB, error) { return db, nil },
		Serve:  func(*http.Server) error { return http.ErrServerClosed },
	}
	err := Run(context.Background(), config.Config{}, slog.Default(), deps)
	require.ErrorContains(t, err, "close failed")
}

func TestRunProtectsProfileRoutesWithJWT(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose()
	var rec *httptest.ResponseRecorder
	deps := Deps{
		OpenDB: func(string) (*sql.DB, error) { return db, nil },
		Serve: func(srv *http.Server) error {
			rec = httptest.NewRecorder()
			srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me", nil))
			return http.ErrServerClosed
		},
	}
	require.NoError(t, Run(context.Background(), config.Config{JWTSecret: "secret"}, slog.Default(), deps))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRunRecordsRoutePatternInHTTPMetrics(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectPing()
	expectMigration(mock)
	mock.ExpectClose()
	scrape := httptest.NewRecorder()
	deps := Deps{
		OpenDB: func(string) (*sql.DB, error) { return db, nil },
		Serve: func(srv *http.Server) error {
			srv.Handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/me", nil))
			srv.Handler.ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			return http.ErrServerClosed
		},
	}
	require.NoError(t, Run(context.Background(), config.Config{JWTSecret: "secret"}, slog.Default(), deps))
	require.Contains(t, scrape.Body.String(), `route="GET /me"`)
}

func TestShutdownRunningServer(t *testing.T) {
	srv := NewServer("0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, Shutdown(srv))
}
