package config

import (
	"errors"
	"os"
	"strings"
)

var ErrJWTSecretRequired = errors.New("JWT_SECRET is required")

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	OTLPEndpoint string
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://identity:identity@localhost:5432/identity?sslmode=disable"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		OTLPEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, ErrJWTSecretRequired
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
