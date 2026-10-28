package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/handler"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/platform/messaging"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/repository"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-task/internal/service"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
)

type Channel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	NotifyReturn(returned chan amqp.Return) chan amqp.Return
	Close() error
}

type Conn interface {
	Channel() (Channel, error)
	Close() error
	IsClosed() bool
}

type Dialer func(url string) (Conn, error)

type Deps struct {
	OpenDB     func(dsn string) (*sql.DB, error)
	DialRabbit Dialer
}

type DB interface {
	PingContext(ctx context.Context) error
	Close() error
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger, deps Deps) error {
	db, err := deps.OpenDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()
	maxOpen := cfg.DBMaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := cfg.DBMaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := WaitForDB(ctx, db); err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}
	if err := repository.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	conn, err := connect(ctx, deps.DialRabbit, cfg.RabbitURL)
	if err != nil {
		return fmt.Errorf("rabbitmq unavailable: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	pub := messaging.New(ch)
	if err := pub.Setup(); err != nil {
		return fmt.Errorf("setup publisher: %w", err)
	}

	repo := repository.NewPostgresTaskRepository(db)
	svc := service.New(repo, nil)
	h := handler.NewWithLogger(svc, map[string]func(context.Context) error{
		"postgres": db.PingContext,
		"rabbitmq": func(context.Context) error {
			if conn.IsClosed() {
				return errors.New("rabbitmq connection closed")
			}
			return nil
		},
	}, log)

	mux := middleware.Correlation(middleware.Logging(log)(otelhttp.NewHandler(h.Routes(cfg.JWTSecret), "service-task.http")))
	srv := NewServer(cfg.Port, mux)

	dispatchCtx, stopDispatcher := context.WithCancel(ctx)
	var dispatcherWG sync.WaitGroup
	dispatcherWG.Add(1)
	go func() {
		defer dispatcherWG.Done()
		messaging.NewDispatcher(repo, pub, log).Run(dispatchCtx)
	}()
	defer func() {
		stopDispatcher()
		dispatcherWG.Wait()
	}()

	serverErr := make(chan error, 1)
	go func() {
		log.Info("service-task listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("serve HTTP: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		return Shutdown(srv)
	case err := <-serverErr:
		return err
	}
}

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

func Shutdown(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

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

func connect(ctx context.Context, dial Dialer, url string) (Conn, error) {
	var lastErr error
	for {
		conn, err := dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, lastErr
		case <-time.After(time.Second):
		}
	}
}
