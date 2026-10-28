package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthOK(t *testing.T) {
	h := NewHealth(map[string]func(context.Context) error{
		"rabbitmq": func(context.Context) error { return nil },
	})
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestHealthDegraded(t *testing.T) {
	h := NewHealth(map[string]func(context.Context) error{
		"rabbitmq": func(context.Context) error { return errors.New("down") },
	})
	rec := httptest.NewRecorder()
	h.Health(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), `"degraded"`)
}
