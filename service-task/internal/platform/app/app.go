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
	platformmetrics "github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/metrics"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
)

// Channel abstrai o canal AMQP usado pelo serviço.
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

// Conn abstrai a conexão AMQP usada na inicialização.
type Conn interface {
	Channel() (Channel, error)
	Close() error
	IsClosed() bool
}

// Dialer abre uma conexão AMQP a partir da URL informada.
type Dialer func(url string) (Conn, error)

// Deps injeta as funções de infraestrutura usadas por Run.
type Deps struct {
	OpenDB     func(dsn string) (*sql.DB, error)
	DialRabbit Dialer
}

// DB abstrai o acesso mínimo ao banco usado na inicialização.
type DB interface {
	PingContext(ctx context.Context) error
	Close() error
}

// Run conecta ao banco e ao RabbitMQ, inicia o dispatcher do outbox e serve HTTP.
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
	svc := service.New(repo)
	metricSet := platformmetrics.New("service-task")
	taskMetrics := metricSet.Counter(
		"task_manager_task_operations_total",
		"Total de operações de tarefa por operação e resultado.",
		"operation", "outcome",
	)
	outboxMetrics := metricSet.Counter(
		"task_manager_task_outbox_events_total",
		"Total de eventos do outbox por resultado do dispatch.",
		"outcome",
	)
	h := handler.New(svc, map[string]func(context.Context) error{
		"postgres": db.PingContext,
		"rabbitmq": func(context.Context) error {
			if conn.IsClosed() {
				return errors.New("rabbitmq connection closed")
			}
			return nil
		},
	}, log).WithTaskMetrics(taskMetrics)

	routes := http.NewServeMux()
	routes.Handle("GET /metrics", metricSet.Handler())
	routes.Handle("/", h.Routes(cfg.JWTSecret))
	mux := middleware.Correlation(middleware.Logging(log)(metricSet.Middleware(otelhttp.NewHandler(routes, "service-task.http"))))
	srv := NewServer(cfg.Port, mux)

	dispatchCtx, stopDispatcher := context.WithCancel(ctx)
	var dispatcherWG sync.WaitGroup
	dispatcherWG.Add(1)
	go func() {
		defer dispatcherWG.Done()
		messaging.NewDispatcher(repo, pub, log, outboxMetrics).Run(dispatchCtx)
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
