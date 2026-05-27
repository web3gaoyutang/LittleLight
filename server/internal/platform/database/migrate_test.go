package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationFilesResolvesPreferredDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "001_init.sql")
	if err := os.WriteFile(file, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	files, resolved, err := migrationFiles(dir)
	if err != nil {
		t.Fatalf("migration files: %v", err)
	}
	if resolved != dir {
		t.Fatalf("expected %s, got %s", dir, resolved)
	}
	if len(files) != 1 || files[0] != file {
		t.Fatalf("unexpected files: %+v", files)
	}
}
