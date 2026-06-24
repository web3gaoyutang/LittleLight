package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/config"
	httpapi "github.com/web3gaoyutang/littlelight/server/internal/http"
	"github.com/web3gaoyutang/littlelight/server/internal/platform/cache"
	"github.com/web3gaoyutang/littlelight/server/internal/platform/database"
	"github.com/web3gaoyutang/littlelight/server/internal/repository"
	"github.com/web3gaoyutang/littlelight/server/internal/service"
)

func main() {
	cfg := config.Load()
	if err := validateProductionAuthConfig(cfg); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var store repository.Store
	var dashboardCache *cache.DashboardCache
	var readiness []httpapi.DependencyCheck

	if pool, err := database.Connect(ctx, cfg.DatabaseURL); err != nil {
		if cfg.AppEnv == "local" {
			log.Printf("postgres unavailable, using memory store: %v", err)
			store = repository.NewMemoryStore()
		} else {
			log.Fatalf("postgres unavailable: %v", err)
		}
	} else {
		defer pool.Close()
		if err := database.Migrate(ctx, pool, cfg.MigrationsDir); err != nil {
			if cfg.AppEnv == "local" {
				log.Printf("postgres migrations failed, using memory store: %v", err)
				store = repository.NewMemoryStore()
			} else {
				log.Fatalf("postgres migrations failed: %v", err)
			}
		} else {
			store = repository.NewPostgresStore(pool)
			readiness = append(readiness, httpapi.DependencyCheck{Name: "postgres", Check: pool.Ping})
			log.Printf("postgres connected, migrations applied, using persistent store")
		}
	}

	if redisClient, err := cache.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB); err != nil {
		if cfg.AppEnv == "local" {
			log.Printf("redis unavailable, cache disabled: %v", err)
		} else {
			log.Fatalf("redis unavailable: %v", err)
		}
	} else {
		defer redisClient.Close()
		dashboardCache = cache.NewDashboardCache(redisClient, 5*time.Minute)
		readiness = append(readiness, httpapi.DependencyCheck{Name: "redis", Check: func(ctx context.Context) error {
			if err := redisClient.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
			return nil
		}})
		log.Printf("redis connected, dashboard cache enabled")
	}

	ai := service.NewAIService(service.AIOptions{
		Provider: cfg.AIProvider,
		APIKey:   cfg.LLMAPIKey,
		BaseURL:  cfg.LLMBaseURL,
		Model:    cfg.LLMModel,
	})
	server := httpapi.NewServer(store, ai, dashboardCache, readiness...)
	server.ConfigureAuth(httpapi.AuthConfig{
		SessionSecret: cfg.SessionSecret,
		WechatAppID:   cfg.WechatAppID,
		WechatSecret:  cfg.WechatSecret,
		AllowDevUser:  cfg.AllowDevUser,
		AllowMockAuth: cfg.AllowMockAuth,
		CORSOrigins:   cfg.CORSOrigins,
	})
	server.ConfigureDictation(service.NewDictationService(service.DictationOptions{
		AppID:                cfg.XFASR.AppID,
		APIKey:               cfg.XFASR.APIKey,
		APISecret:            cfg.XFASR.APISecret,
		Endpoint:             cfg.XFASR.Endpoint,
		MaxSessionSeconds:    cfg.XFASR.MaxSessionSeconds,
		MaxConcurrentPerUser: cfg.XFASR.MaxConcurrentPerUser,
		DailyLimitPerUser:    cfg.XFASR.DailyLimitPerUser,
	}))
	httpServer := newHTTPServer(cfg.HTTPAddr, server.Routes())
	log.Printf("littlelight api listening on %s", cfg.HTTPAddr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func validateProductionAuthConfig(cfg config.Config) error {
	if !isProductionEnv(cfg.AppEnv) {
		return nil
	}
	if cfg.AllowDevUser {
		return errors.New("AUTH_ALLOW_DEV_USER must be false outside local environment")
	}
	if cfg.AllowMockAuth {
		return errors.New("AUTH_ALLOW_MOCK_LOGIN must be false outside local environment")
	}
	if len(cfg.CORSOrigins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS must be configured outside local environment")
	}
	for _, origin := range cfg.CORSOrigins {
		if origin == "*" {
			return errors.New("CORS_ALLOWED_ORIGINS must not contain * outside local environment")
		}
	}
	secret := strings.TrimSpace(cfg.SessionSecret)
	if len(secret) < 32 || secret == "change-me-in-deploy" {
		return errors.New("SESSION_SECRET must be configured with at least 32 characters outside local environment")
	}
	if strings.TrimSpace(cfg.WechatAppID) == "" || strings.TrimSpace(cfg.WechatSecret) == "" {
		return errors.New("WECHAT_APP_ID and WECHAT_APP_SECRET must be configured outside local environment")
	}
	return nil
}

func isProductionEnv(appEnv string) bool {
	normalized := strings.ToLower(strings.TrimSpace(appEnv))
	return normalized == "prod" || normalized == "production"
}
