package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/config"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return strconv.Itoa(port)
}

func runHealth(t *testing.T, closed bool) int {
	t.Helper()
	db, mock := newMock(t)
	mock.ExpectPing()
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS tasks")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPing()
	conn := &fakeConn{ch: &fakeChannel{}, closed: closed}
	port := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = Run(ctx, config.Config{Port: port}, slog.Default(), okDeps(db, conn))
	}()

	var resp *http.Response
	var err error
	for i := 0; i < 100; i++ {
		resp, err = http.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestRunHealthOK(t *testing.T) {
	require.Equal(t, http.StatusOK, runHealth(t, false))
}

func TestRunHealthDegraded(t *testing.T) {
	require.Equal(t, http.StatusServiceUnavailable, runHealth(t, true))
}
