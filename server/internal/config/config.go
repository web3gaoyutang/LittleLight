package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv        string
	HTTPAddr      string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	MigrationsDir string
	AIProvider    string
	AIAPIKey      string
}

func Load() Config {
	return Config{
		AppEnv:        env("APP_ENV", "local"),
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://littlelight:littlelight@localhost:5432/littlelight?sslmode=disable"),
		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		MigrationsDir: env("MIGRATIONS_DIR", "server/migrations"),
		AIProvider:    env("AI_PROVIDER", "mock"),
		AIAPIKey:      env("AI_API_KEY", ""),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
