package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	LLMAPIKey     string
	LLMBaseURL    string
	LLMModel      string
}

func Load() Config {
	loadDotEnv()

	llmAPIKey := env("LLM_API_KEY", env("AI_API_KEY", ""))
	llmBaseURL := env("LLM_BASE_URL", "")
	aiProvider := env("AI_PROVIDER", "")
	if aiProvider == "" {
		aiProvider = "mock"
		if llmAPIKey != "" && llmBaseURL != "" {
			aiProvider = "llm"
		}
	}
	return Config{
		AppEnv:        env("APP_ENV", "local"),
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://littlelight:littlelight@localhost:5432/littlelight?sslmode=disable"),
		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		MigrationsDir: env("MIGRATIONS_DIR", "server/migrations"),
		AIProvider:    aiProvider,
		AIAPIKey:      env("AI_API_KEY", ""),
		LLMAPIKey:     llmAPIKey,
		LLMBaseURL:    llmBaseURL,
		LLMModel:      env("LLM_MODEL", "gpt-4o-mini"),
	}
}

func loadDotEnv() {
	for _, path := range dotEnvCandidates() {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := parseEnvLine(scanner.Text())
			if !ok || os.Getenv(key) != "" {
				continue
			}
			_ = os.Setenv(key, value)
		}
		return
	}
}

func dotEnvCandidates() []string {
	return []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}
}

func parseEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.Contains(trimmed, "=") {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, "=", 2)
	key := strings.TrimSpace(parts[0])
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, `"'`)
	return key, value, true
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
