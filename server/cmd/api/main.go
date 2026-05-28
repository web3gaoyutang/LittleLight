package main

import (
	"context"
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

	if pool, err := database.Connect(ctx, cfg.DatabaseURL); err != nil {
		log.Printf("postgres unavailable, using memory store: %v", err)
		store = repository.NewMemoryStore()
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
			log.Printf("postgres connected, migrations applied, using persistent store")
		}
	}

	if redisClient, err := cache.Connect(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB); err != nil {
		log.Printf("redis unavailable, cache disabled: %v", err)
	} else {
		defer redisClient.Close()
		dashboardCache = cache.NewDashboardCache(redisClient, 5*time.Minute)
		log.Printf("redis connected, dashboard cache enabled")
	}

	ai := service.NewAIService(service.AIOptions{
		Provider: cfg.AIProvider,
		APIKey:   cfg.LLMAPIKey,
		BaseURL:  cfg.LLMBaseURL,
		Model:    cfg.LLMModel,
	})
	server := httpapi.NewServer(store, ai, dashboardCache)
	log.Printf("littlelight api listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
