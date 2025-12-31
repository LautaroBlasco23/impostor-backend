package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	RedisURL       string
	PostgresURL    string
	AllowedOrigins string
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "3000"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		PostgresURL:    getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/gamedb?sslmode=disable"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
