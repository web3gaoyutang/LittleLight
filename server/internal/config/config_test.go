package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")

	cfg := Load()

	if cfg.AppEnv != "local" || cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected default app config: %+v", cfg)
	}
	if cfg.RedisAddr != "localhost:6379" || cfg.RedisDB != 0 {
		t.Fatalf("unexpected default redis config: %+v", cfg)
	}
	if cfg.MigrationsDir != "server/migrations" {
		t.Fatalf("unexpected default migrations dir: %s", cfg.MigrationsDir)
	}
	if cfg.AIProvider != "mock" || cfg.LLMModel != "gpt-4o-mini" {
		t.Fatalf("unexpected default AI config: %+v", cfg)
	}
}

func TestLoadAutoSelectsLLMWhenCredentialsExist(t *testing.T) {
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_BASE_URL", "https://llm.example.com")

	cfg := Load()

	if cfg.AIProvider != "llm" {
		t.Fatalf("expected llm provider, got %s", cfg.AIProvider)
	}
	if cfg.LLMAPIKey != "test-key" || cfg.LLMBaseURL != "https://llm.example.com" {
		t.Fatalf("unexpected LLM config: %+v", cfg)
	}
}

func TestLoadKeepsExplicitAIProviderAndLegacyKeyFallback(t *testing.T) {
	t.Setenv("AI_PROVIDER", "qwen")
	t.Setenv("AI_API_KEY", "legacy-key")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_BASE_URL", "https://llm.example.com")

	cfg := Load()

	if cfg.AIProvider != "qwen" {
		t.Fatalf("expected explicit provider, got %s", cfg.AIProvider)
	}
	if cfg.LLMAPIKey != "legacy-key" || cfg.AIAPIKey != "legacy-key" {
		t.Fatalf("expected legacy key fallback, got %+v", cfg)
	}
}

func TestLoadRedisDBFallback(t *testing.T) {
	t.Setenv("REDIS_DB", "not-a-number")
	if cfg := Load(); cfg.RedisDB != 0 {
		t.Fatalf("invalid redis db should fall back to zero, got %d", cfg.RedisDB)
	}

	t.Setenv("REDIS_DB", "3")
	if cfg := Load(); cfg.RedisDB != 3 {
		t.Fatalf("expected configured redis db, got %d", cfg.RedisDB)
	}
}

func TestLoadReadsDotEnvWithoutOverridingExplicitEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", ":19090")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_MODEL", "")

	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte(`
APP_ENV=docker
HTTP_ADDR=:18080
REDIS_DB=4
LLM_API_KEY='dotenv-key'
LLM_BASE_URL="https://dotenv.example.com"
`), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}

	cfg := Load()

	if cfg.AppEnv != "docker" {
		t.Fatalf("expected APP_ENV from .env, got %s", cfg.AppEnv)
	}
	if cfg.HTTPAddr != ":19090" {
		t.Fatalf("explicit HTTP_ADDR should win over .env, got %s", cfg.HTTPAddr)
	}
	if cfg.RedisDB != 4 {
		t.Fatalf("expected Redis DB from .env, got %d", cfg.RedisDB)
	}
	if cfg.AIProvider != "llm" || cfg.LLMAPIKey != "dotenv-key" || cfg.LLMBaseURL != "https://dotenv.example.com" {
		t.Fatalf("expected LLM config from .env, got %+v", cfg)
	}
}
