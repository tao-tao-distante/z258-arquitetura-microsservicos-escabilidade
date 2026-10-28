package config

import "os"

type Config struct {
	Port         string
	RabbitURL    string
	OTLPEndpoint string
}

func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		RabbitURL:    getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		OTLPEndpoint: otlpEndpoint(),
	}
}

func otlpEndpoint() string {
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
