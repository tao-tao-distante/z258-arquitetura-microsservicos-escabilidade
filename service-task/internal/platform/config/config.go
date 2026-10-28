package config

import (
	"errors"
	"os"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RabbitURL      string
	JWTSecret      string
	OTLPEndpoint   string
	DBMaxOpenConns int
	DBMaxIdleConns int
}

func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://task:task@localhost:5432/task?sslmode=disable"),
		RabbitURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		DBMaxOpenConns: 25,
		DBMaxIdleConns: 10,
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
