package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

type DashboardCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewDashboardCache(client *redis.Client, ttl time.Duration) *DashboardCache {
	if client == nil {
		return nil
	}
	return &DashboardCache{client: client, ttl: ttl}
}

func (c *DashboardCache) Get(ctx context.Context, userID domain.ID, day time.Time) (domain.DashboardSummary, bool) {
	if c == nil || c.client == nil {
		return domain.DashboardSummary{}, false
	}
	raw, err := c.client.Get(ctx, c.key(userID, day)).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Printf("dashboard cache get failed: %v", err)
		}
		return domain.DashboardSummary{}, false
	}
	var summary domain.DashboardSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		log.Printf("dashboard cache decode failed: %v", err)
		return domain.DashboardSummary{}, false
	}
	return summary, true
}

func (c *DashboardCache) Set(ctx context.Context, userID domain.ID, day time.Time, summary domain.DashboardSummary) {
	if c == nil || c.client == nil {
		return
	}
	payload, err := json.Marshal(summary)
	if err != nil {
		log.Printf("dashboard cache encode failed: %v", err)
		return
	}
	if err := c.client.Set(ctx, c.key(userID, day), payload, c.ttl).Err(); err != nil {
		log.Printf("dashboard cache set failed: %v", err)
	}
}

func (c *DashboardCache) Invalidate(ctx context.Context, userID domain.ID, day time.Time) {
	if c == nil || c.client == nil {
		return
	}
	if err := c.client.Del(ctx, c.key(userID, day)).Err(); err != nil {
		log.Printf("dashboard cache invalidate failed: %v", err)
	}
}

func (c *DashboardCache) key(userID domain.ID, day time.Time) string {
	return fmt.Sprintf("dashboard:%s:%s", userID, day.Format("2006-01-02"))
}
