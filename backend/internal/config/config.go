package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	AppEnv      string
	CORSOrigins []string
}

func Load() Config {
	return Config{
		HTTPAddr:    env("HTTP_ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://studytrack:studytrack@localhost:5432/studytrack?sslmode=disable"),
		JWTSecret:   env("JWT_SECRET", "development-only-change-me"),
		AppEnv:      env("APP_ENV", "development"),
		CORSOrigins: splitCSV(env("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")),
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
