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
	SessionSecret string
	WechatAppID   string
	WechatSecret  string
	AllowDevUser  bool
	AllowMockAuth bool
	CORSOrigins   []string
	AIProvider    string
	AIAPIKey      string
	LLMAPIKey     string
	LLMBaseURL    string
	LLMModel      string
	XFASR         XFASRConfig
}

type XFASRConfig struct {
	AppID                string
	APIKey               string
	APISecret            string
	Endpoint             string
	MaxSessionSeconds    int
	MaxConcurrentPerUser int
	DailyLimitPerUser    int
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
	appEnv := env("APP_ENV", "local")
	return Config{
		AppEnv:        appEnv,
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://littlelight:littlelight@localhost:5432/littlelight?sslmode=disable"),
		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),
		MigrationsDir: env("MIGRATIONS_DIR", "server/migrations"),
		SessionSecret: env("SESSION_SECRET", ""),
		WechatAppID:   env("WECHAT_APP_ID", ""),
		WechatSecret:  env("WECHAT_APP_SECRET", ""),
		AllowDevUser:  envBool("AUTH_ALLOW_DEV_USER", isLocalEnv(appEnv)),
		AllowMockAuth: envBool("AUTH_ALLOW_MOCK_LOGIN", !isProductionEnv(appEnv)),
		CORSOrigins:   envList("CORS_ALLOWED_ORIGINS"),
		AIProvider:    aiProvider,
		AIAPIKey:      env("AI_API_KEY", ""),
		LLMAPIKey:     llmAPIKey,
		LLMBaseURL:    llmBaseURL,
		LLMModel:      env("LLM_MODEL", "gpt-4o-mini"),
		XFASR: XFASRConfig{
			AppID:                env("XF_ASR_APP_ID", ""),
			APIKey:               env("XF_ASR_API_KEY", ""),
			APISecret:            env("XF_ASR_API_SECRET", ""),
			Endpoint:             env("XF_ASR_ENDPOINT", "wss://iat-api.xfyun.cn/v2/iat"),
			MaxSessionSeconds:    envInt("XF_ASR_MAX_SESSION_SECONDS", 55),
			MaxConcurrentPerUser: envInt("XF_ASR_MAX_CONCURRENT_PER_USER", 1),
			DailyLimitPerUser:    envInt("XF_ASR_DAILY_LIMIT_PER_USER", 80),
		},
	}
}

func isLocalEnv(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "local")
}

func isProductionEnv(appEnv string) bool {
	normalized := strings.ToLower(strings.TrimSpace(appEnv))
	return normalized == "prod" || normalized == "production"
}

func envList(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
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

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
