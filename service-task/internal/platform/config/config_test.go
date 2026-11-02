package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	require.Error(t, err)
	// JWT_SECRET must never have a development fallback.
	t.Setenv("JWT_SECRET", "secret")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "8080", c.Port)
	require.Contains(t, c.DatabaseURL, "task")
	require.Contains(t, c.RabbitURL, "amqp")
	require.Equal(t, "secret", c.JWTSecret)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("DATABASE_URL", "url")
	t.Setenv("RABBITMQ_URL", "amqp://x")
	t.Setenv("JWT_SECRET", "secret")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "9999", c.Port)
	require.Equal(t, "url", c.DatabaseURL)
	require.Equal(t, "amqp://x", c.RabbitURL)
	require.Equal(t, "secret", c.JWTSecret)
}

func TestLoadOTLPEndpoint(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")
	c, err := Load()
	require.NoError(t, err)
	require.Equal(t, "otel-collector:4317", c.OTLPEndpoint)
}
