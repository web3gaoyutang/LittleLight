package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresStoreFindOrCreateUserByWechatOpenID(t *testing.T) {
	databaseURL := os.Getenv("LITTLELIGHT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set LITTLELIGHT_TEST_DATABASE_URL to run postgres repository integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	store := NewPostgresStore(pool)
	openID := "test-openid-upsert"
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE wechat_open_id = $1`, openID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE wechat_open_id = $1`, openID)
	})

	first, err := store.FindOrCreateUserByWechatOpenID(ctx, openID, "第一次登录", "")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, err := store.FindOrCreateUserByWechatOpenID(ctx, openID, "第二次登录", "")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if first.ID == "" || second.ID != first.ID {
		t.Fatalf("expected same user for same openid, first=%+v second=%+v", first, second)
	}
	if second.Name != "第二次登录" {
		t.Fatalf("expected upsert to update profile name, got %+v", second)
	}
}
