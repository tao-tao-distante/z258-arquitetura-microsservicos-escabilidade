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
