package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/correlation"
	"github.com/tao-tao-distante/z258-arquitetura-microsservicos-escabilidade/shared/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HeaderCorrelationID é o cabeçalho HTTP que transporta o correlation ID.
const HeaderCorrelationID = "x-correlation-id"

const requestBodyCaptureLimit = 64 * 1024

var loggedBodyFields = map[string]struct{}{
	"after_id":  {},
	"before_id": {},
	"due_date":  {},
	"priority":  {},
	"status":    {},
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

type requestBodyCapture struct {
	io.ReadCloser
	buffer        bytes.Buffer
	bytesRead     int64
	contentLength int64
}

func (c *requestBodyCapture) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	c.bytesRead += int64(n)
	if remaining := requestBodyCaptureLimit - c.buffer.Len(); remaining > 0 {
		_, _ = c.buffer.Write(p[:min(n, remaining)])
	}
	return n, err
}

func (c *requestBodyCapture) truncated() bool {
	return c.bytesRead > requestBodyCaptureLimit || c.contentLength > requestBodyCaptureLimit
}

// WriteHeader registra o status e encaminha a chamada ao ResponseWriter.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Correlation aceita apenas UUID v4 e gera um novo quando o header e o contexto são inválidos.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderCorrelationID)
		if !correlation.ValidID(id) {
			id = correlation.FromContext(r.Context())
		}
		if !correlation.ValidID(id) {
			id = correlation.NewID()
		}
		ctx := correlation.NewContext(r.Context(), id)
		trace.SpanFromContext(ctx).SetAttributes(attribute.String("correlation_id", id))
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
			var body *requestBodyCapture
			if r.Body != nil && r.Body != http.NoBody {
				body = &requestBodyCapture{ReadCloser: r.Body, contentLength: r.ContentLength}
				r.Body = body
			}
			next.ServeHTTP(rec, r)
			latency := time.Since(start).Milliseconds()
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", rec.status,
				"latency_ms", latency,
			}
			attrs = appendRequestBodyAttrs(attrs, r.Header.Get("Content-Type"), body)
			logger.WithContext(r.Context(), l).InfoContext(r.Context(), "http_request", attrs...)
		})
	}
}

func appendRequestBodyAttrs(attrs []any, contentType string, body *requestBodyCapture) []any {
	if body == nil {
		return attrs
	}
	truncated := body.truncated()
	attrs = append(attrs,
		"request_body_bytes", body.bytesRead,
		"request_body_truncated", truncated,
	)
	if !isJSON(contentType) || body.bytesRead == 0 {
		return attrs
	}
	if truncated {
		return append(attrs, "request_body_valid_json", false)
	}

	var value any
	if err := json.Unmarshal(body.buffer.Bytes(), &value); err != nil {
		return append(attrs, "request_body_valid_json", false)
	}
	return append(attrs,
		"request_body_valid_json", true,
		"request_body", sanitizeBody(value),
	)
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"))
}

func sanitizeBody(value any) any {
	object, ok := value.(map[string]any)
	if !ok {
		return "[REDACTED]"
	}
	sanitized := make(map[string]any, len(object))
	for key, fieldValue := range object {
		_, allowed := loggedBodyFields[strings.ToLower(key)]
		if allowed && isScalar(fieldValue) {
			sanitized[key] = fieldValue
			continue
		}
		sanitized[key] = "[REDACTED]"
	}
	return sanitized
}

func isScalar(value any) bool {
	switch value.(type) {
	case nil, bool, float64, string:
		return true
	default:
		return false
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
	onLimit func(*http.Request)
}

// NewRateLimiter cria um limitador com limite, janela e relógio informados.
func NewRateLimiter(limit int, window time.Duration, now func() time.Time) *RateLimiter {
	return &RateLimiter{clients: make(map[string]rateWindow), limit: limit, window: window, now: now}
}

// OnLimit configura a função chamada quando uma requisição é rejeitada.
func (l *RateLimiter) OnLimit(callback func(*http.Request)) *RateLimiter {
	l.onLimit = callback
	return l
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
			if l.onLimit != nil {
				l.onLimit(r)
			}
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
