package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("RABBITMQ_URL", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	c := Load()
	require.Equal(t, "8080", c.Port)
	require.Contains(t, c.RabbitURL, "amqp")
	require.Empty(t, c.OTLPEndpoint)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("RABBITMQ_URL", "amqp://x")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4317")
	c := Load()
	require.Equal(t, "9999", c.Port)
	require.Equal(t, "amqp://x", c.RabbitURL)
	require.Equal(t, "http://collector:4317", c.OTLPEndpoint)
}

func TestLoadUsesTraceSpecificOTLPEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://traces:4317")
	require.Equal(t, "http://traces:4317", Load().OTLPEndpoint)
}
