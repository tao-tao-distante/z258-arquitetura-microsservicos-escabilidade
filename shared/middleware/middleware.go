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

const HeaderCorrelationID = "x-correlation-id"

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderCorrelationID)
		if id == "" {
			id = correlation.FromContext(r.Context())
		}
		if id == "" {
			id = correlation.NewID()
		}
		ctx := correlation.NewContext(r.Context(), id)
		w.Header().Set(HeaderCorrelationID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CorrelationUUID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderCorrelationID)
		if _, err := uuid.Parse(id); err != nil {
			id = correlation.NewID()
		}
		ctx := correlation.NewContext(r.Context(), id)
		w.Header().Set(HeaderCorrelationID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]rateWindow
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	return &RateLimiter{clients: make(map[string]rateWindow), limit: limit, window: window, now: now}
}

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
