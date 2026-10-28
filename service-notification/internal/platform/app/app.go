package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/handler"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/service-notification/internal/platform/config"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/middleware"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Channel abstrai o canal AMQP usado pelo serviço.
type Channel interface {
	handler.Channel
	Close() error
	IsClosed() bool
}

// Conn abstrai a conexão AMQP usada na inicialização.
type Conn interface {
	Channel() (Channel, error)
	Close() error
	IsClosed() bool
}

// Dialer abre uma conexão AMQP a partir da URL informada.
type Dialer func(url string) (Conn, error)

// Run conecta ao RabbitMQ, inicia o consumidor e serve o endpoint de saúde.
func Run(ctx context.Context, cfg config.Config, log *slog.Logger, dial Dialer) error {
	shutdownTelemetry, _, err := telemetry.Setup(ctx, cfg.OTLPEndpoint, "service-notification")
	if err != nil {
		return fmt.Errorf("setup telemetry: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			log.Error("telemetry shutdown failed", "error", err)
		}
	}()

	conn, err := connect(ctx, dial, cfg.RabbitURL)
	if err != nil {
		return fmt.Errorf("rabbitmq unavailable: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	consumer := handler.NewConsumer(ch, log)
	consumerDone := make(chan error, 1)
	go func() { consumerDone <- consumer.Start(runCtx) }()

	health := newHealth(conn, ch, consumer)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Health)
	httpHandler := otelhttp.NewHandler(middleware.CorrelationUUID(middleware.Logging(log)(mux)), "http.server")
	srv := NewServer(cfg.Port, httpHandler)
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		cancel()
		<-consumerDone
		return fmt.Errorf("http server listen: %w", err)
	}
	serverDone := make(chan error, 1)
	go func() {
		log.Info("service-notification listening", "port", cfg.Port)
		err := srv.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverDone <- err
	}()

	var runErr error
	consumerFinished := false
	serverFinished := false
	select {
	case <-ctx.Done():
	case err := <-consumerDone:
		consumerFinished = true
		if err != nil {
			runErr = fmt.Errorf("consumer: %w", err)
		}
	case err := <-serverDone:
		serverFinished = true
		if err != nil {
			runErr = fmt.Errorf("http server: %w", err)
		}
	}

	cancel()
	shutdownErr := Shutdown(srv)
	if !consumerFinished {
		if err := <-consumerDone; runErr == nil && err != nil {
			runErr = fmt.Errorf("consumer: %w", err)
		}
	}
	if !serverFinished {
		if err := <-serverDone; runErr == nil && err != nil {
			runErr = fmt.Errorf("http server: %w", err)
		}
	}
	return errors.Join(runErr, shutdownErr)
}

func newHealth(conn Conn, ch Channel, consumer *handler.Consumer) *handler.Health {
	return handler.NewHealth(map[string]func(context.Context) error{
		"rabbitmq": func(context.Context) error {
			if conn.IsClosed() {
				return errors.New("rabbitmq connection closed")
			}
			if ch.IsClosed() {
				return errors.New("rabbitmq channel closed")
			}
			if !consumer.Active() {
				return errors.New("rabbitmq consumer inactive")
			}
			return nil
		},
	})
}

// NewServer cria o servidor HTTP com timeouts explícitos.
func NewServer(port string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Shutdown encerra o servidor HTTP com timeout de 10 segundos.
func Shutdown(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
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
