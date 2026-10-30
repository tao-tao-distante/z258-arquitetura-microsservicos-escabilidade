package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddlewareRecordsNormalizedRouteStatusAndDuration(t *testing.T) {
	set := New("service-test")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	rec := httptest.NewRecorder()
	set.Middleware(mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/42", nil))

	require.Equal(t, float64(1), testutil.ToFloat64(set.requests.WithLabelValues(
		"service-test", http.MethodGet, "GET /tasks/{id}", "201",
	)))
	require.Positive(t, testutil.CollectAndCount(set.duration))
}

func copyRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}

func TestHTTPMiddlewareRecordsRoutePatternBehindRequestCopies(t *testing.T) {
	inner := http.NewServeMux()
	inner.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	outer := http.NewServeMux()
	outer.Handle("/", copyRequest(RoutePattern(inner)))

	tests := []struct {
		name    string
		handler http.Handler
	}{
		{name: "copia antes do mux interno", handler: copyRequest(RoutePattern(inner))},
		{name: "mux externo com rota curinga", handler: copyRequest(outer)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := New("service-test")
			rec := httptest.NewRecorder()
			set.Middleware(tt.handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/42", nil))

			require.Equal(t, float64(1), testutil.ToFloat64(set.requests.WithLabelValues(
				"service-test", http.MethodGet, "GET /tasks/{id}", "200",
			)))
		})
	}
}

func TestHTTPMiddlewareRecordsUnmatchedWhenNoPatternIsKnown(t *testing.T) {
	set := New("service-test")
	handler := copyRequest(RoutePattern(http.NotFoundHandler()))
	rec := httptest.NewRecorder()
	set.Middleware(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	require.Equal(t, float64(1), testutil.ToFloat64(set.requests.WithLabelValues(
		"service-test", http.MethodGet, "unmatched", "404",
	)))
}

func TestRoutePatternServesWithoutMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	RoutePattern(mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHTTPMiddlewareDoesNotRecordMetricsScrapes(t *testing.T) {
	set := New("service-test")
	rec := httptest.NewRecorder()
	set.Middleware(set.Handler()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Zero(t, testutil.CollectAndCount(set.requests))
}

func TestCounterRegistersCustomMetric(t *testing.T) {
	set := New("service-test")
	counter := set.Counter("task_manager_test_operations_total", "Test operations.", "outcome")
	counter.Inc("success")

	require.Equal(t, float64(1), testutil.ToFloat64(counter.vec.WithLabelValues("success")))
}
