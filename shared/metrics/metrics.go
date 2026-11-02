// Package metrics fornece instrumentação Prometheus comum aos serviços.
package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Set mantém um registro isolado e as métricas HTTP de um serviço.
type Set struct {
	service  string
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// Counter encapsula um contador vetorial com labels de baixa cardinalidade.
type Counter struct{ vec *prometheus.CounterVec }

// New cria métricas HTTP RED e inclui métricas padrão de processo e runtime Go.
func New(service string) *Set {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "task_manager_http_requests_total",
		Help: "Total de requisições HTTP processadas.",
	}, []string{"service", "method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "task_manager_http_request_duration_seconds",
		Help:    "Duração das requisições HTTP em segundos.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"service", "method", "route", "status"})
	registry.MustRegister(requests, duration)
	return &Set{service: service, registry: registry, requests: requests, duration: duration}
}

// Counter registra um contador de negócio no registro do serviço.
func (s *Set) Counter(name, help string, labels ...string) *Counter {
	vec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, labels)
	s.registry.MustRegister(vec)
	return &Counter{vec: vec}
}

// Inc incrementa o contador para a combinação de labels informada.
func (c *Counter) Inc(labels ...string) { c.vec.WithLabelValues(labels...).Inc() }

// Handler expõe o registro no formato de texto do Prometheus.
func (s *Set) Handler() http.Handler {
	return promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
}

// Middleware registra taxa, status e duração usando o padrão da rota, nunca a URL bruta.
func (s *Set) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		route := new(string)
		r = r.WithContext(context.WithValue(r.Context(), routeKey{}, route))
		next.ServeHTTP(recorder, r)
		if *route == "" {
			*route = r.Pattern
		}
		if *route == "" {
			*route = "unmatched"
		}
		labels := []string{s.service, r.Method, *route, strconv.Itoa(recorder.status)}
		s.requests.WithLabelValues(labels...).Inc()
		s.duration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())
	})
}

type routeKey struct{}

func RoutePattern(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		if route, ok := r.Context().Value(routeKey{}).(*string); ok && r.Pattern != "" {
			*route = r.Pattern
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wrote {
		return
	}
	r.wrote = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
