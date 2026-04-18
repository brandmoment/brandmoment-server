package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port              string
	DatabaseURL       string
	BetterAuthJWKSURL string
	OTLPEndpoint      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		BetterAuthJWKSURL: os.Getenv("BETTERAUTH_JWKS_URL"),
		OTLPEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BetterAuthJWKSURL == "" {
		return nil, fmt.Errorf("BETTERAUTH_JWKS_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
