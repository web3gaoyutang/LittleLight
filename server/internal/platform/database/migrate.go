package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	files, _, err := migrationFiles(migrationsDir)
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filepath.Base(file), err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("run migration %s: %w", filepath.Base(file), err)
		}
	}
	return nil
}

func migrationFiles(preferredDir string) ([]string, string, error) {
	candidates := []string{preferredDir, "migrations", "server/migrations", "/app/migrations"}
	seen := make(map[string]bool, len(candidates))
	for _, dir := range candidates {
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
		if err != nil {
			return nil, "", err
		}
		sort.Strings(files)
		if len(files) > 0 {
			return files, dir, nil
		}
	}
	return nil, preferredDir, nil
}
