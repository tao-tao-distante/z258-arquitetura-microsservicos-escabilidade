package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/handler"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-identity/internal/service"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Deps injeta as funções de infraestrutura usadas por Run.
type Deps struct {
	OpenDB func(dsn string) (*sql.DB, error)
	Serve  func(*http.Server) error
}

// DB abstrai o acesso mínimo ao banco usado na inicialização.
type DB interface {
	PingContext(ctx context.Context) error
	Close() error
}

// Run conecta ao banco, migra, monta o servidor e aguarda encerramento gracioso.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger, deps Deps) (runErr error) {
	db, err := deps.OpenDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close database: %w", err))
		}
	}()
	ConfigureDBPool(db)

	if err := WaitForDB(ctx, db); err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}
	if err := repository.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	repo := repository.NewPostgresUserRepository(db)
	svc := service.New(repo, cfg.JWTSecret, time.Hour)
	h := handler.New(svc, map[string]func(context.Context) error{
		"postgres": db.PingContext,
	}, log)

	mux := middleware.Correlation(middleware.Logging(log)(h.Routes()))
	mux = otelhttp.NewHandler(mux, "service-identity")
	srv := NewServer(cfg.Port, mux)
	serve := deps.Serve
	if serve == nil {
		serve = func(server *http.Server) error { return server.ListenAndServe() }
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("service-identity listening", "port", cfg.Port)
		serveErr <- serve(srv)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		if err := Shutdown(srv); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	}
}

// NewServer cria o servidor HTTP com timeouts explícitos.
func NewServer(port string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// ConfigureDBPool aplica limites de conexão e tempos de vida ao pool.
func ConfigureDBPool(db *sql.DB) {
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
}

// Shutdown encerra o servidor HTTP com timeout de 10 segundos.
func Shutdown(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// WaitForDB tenta pingar o banco até obter sucesso ou o contexto ser cancelado.
func WaitForDB(ctx context.Context, db DB) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
