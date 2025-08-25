package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	dbfs "github.com/garnizeh/rag/db"
	"github.com/garnizeh/rag/internal/config"
	"github.com/garnizeh/rag/internal/db"
)

// TestMigrateOnStart_TempWorkdir ensures the migration runner works when migrations
// are present in the repository layout, but keeps all files inside a temporary
// working directory so the real repository is not modified.
func TestMigrateOnStart_TempWorkdir(t *testing.T) {
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "rag-startup-test-")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	// ensure we cleanup
	defer os.RemoveAll(tmpDir)

	// prepare DB path inside tmpDir (we keep working dir as repo root so
	// db.Migrate reads the official repo db/migrations)
	dbPath := filepath.Join(tmpDir, "test.db")

	// minimal config enabling migrations
	cfgY := "addr: \":0\"\n" +
		"database_path: '" + dbPath + "'\n" +
		"migrate_on_start: true\n" +
		"engine:\n  model: \"test-model\"\n"

	cfgPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgY), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// allow insecure default JWTSecret for this test
	prevEnv := os.Getenv("RAG_ENV")
	_ = os.Setenv("RAG_ENV", "development")
	defer func() {
		_ = os.Setenv("RAG_ENV", prevEnv)
	}()

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	dbCtx, dbCancel := context.WithTimeout(ctx, cfg.APITimeout)
	defer dbCancel()

	d, err := db.New(dbCtx, cfg.DatabasePath, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	// Run migration runner (this will read repo db/migrations via embedded FS)
	if err := db.Migrate(dbCtx, d, dbfs.Migrations, dbfs.SeedFiles); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// verify schema_migrations has at least one entry
	var count int
	row := d.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations`)
	if row == nil {
		t.Fatalf("query row is nil")
	}
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_migrations count: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected migrations recorded, got 0")
	}
}
