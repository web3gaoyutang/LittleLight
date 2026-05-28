package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	log.Printf("littlelight api listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
