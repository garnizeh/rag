package db_test

import (
	"context"
	"testing"

	dbfs "github.com/garnizeh/rag/db"
	"github.com/garnizeh/rag/internal/db"
)

// Note: this test uses an in-memory sqlite database and a temporary migrations
// directory to validate idempotent behavior of Migrate.
func TestMigrate_Idempotent(t *testing.T) {
	ctx := context.Background()

	// create in-memory DB
	d, err := db.New(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer d.Close()

	// Run Migrate using the embedded migrations and seed files included in package db
	if err := db.Migrate(ctx, d, dbfs.Migrations, dbfs.SeedFiles); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Run again to ensure idempotency
	if err := db.Migrate(ctx, d, dbfs.Migrations, dbfs.SeedFiles); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}

	// verify schema_migrations has at least one entry (embedded migrations applied)
	var count int
	row := d.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations`)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_migrations count: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least 1 migration recorded, got %d", count)
	}

	// verify a known table from the embedded migrations exists (engineers)
	var name string
	r1 := d.QueryRow(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='engineers'`)
	if err := r1.Scan(&name); err != nil {
		t.Fatalf("expected engineers table exists: %v", err)
	}
}
