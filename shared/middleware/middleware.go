package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
)

// HeaderCorrelationID é o cabeçalho HTTP que transporta o correlation ID.
const HeaderCorrelationID = "x-correlation-id"

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader registra o status e encaminha a chamada ao ResponseWriter.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Correlation aceita apenas UUID válido e gera um novo quando o header e o contexto são inválidos.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderCorrelationID)
		if _, err := uuid.Parse(id); err != nil {
			id = correlation.FromContext(r.Context())
		}
		if _, err := uuid.Parse(id); err != nil {
			id = correlation.NewID()
		}
		ctx := correlation.NewContext(r.Context(), id)
		w.Header().Set(HeaderCorrelationID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging registra método, rota, status e latência de cada requisição.
func Logging(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			latency := time.Since(start).Milliseconds()
			logger.WithCorrelation(r.Context(), l).InfoContext(r.Context(), "http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", rec.status,
				"latency_ms", latency,
			)
		})
	}
}

type rateWindow struct {
	started time.Time
	count   int
}

// RateLimiter limita requisições por cliente dentro de uma janela fixa.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]rateWindow
	limit   int
	window  time.Duration
	now     func() time.Time
}

// NewRateLimiter cria um limitador com limite, janela e relógio informados.
func NewRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	return &RateLimiter{clients: make(map[string]rateWindow), limit: limit, window: window, now: now}
}

// Handler rejeita com 429 quando o cliente excede o limite da janela.
func (l *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		now := l.now()
		l.mu.Lock()
		entry := l.clients[host]
		if entry.started.IsZero() || now.Sub(entry.started) >= l.window {
			entry = rateWindow{started: now}
		}
		entry.count++
		l.clients[host] = entry
		limited := entry.count > l.limit
		for key, candidate := range l.clients {
			if now.Sub(candidate.started) >= l.window {
				delete(l.clients, key)
			}
		}
		l.mu.Unlock()
		if limited {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
