package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Migrate applies migrations and optional seed files found in the repository.
// It creates a `schema_migrations` table to track applied migrations and applies
// any SQL files in `db/migrations/` that have not yet been recorded. Seed files
// in seedDir are applied idempotently where possible.
func Migrate(ctx context.Context, d *DB, migrationFS embed.FS, seedFS embed.FS) error {
	// ensure migrations table exists
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	// embedded migrations are provided under "migrations/..." in the top-level db package
	migDir := "migrations"

	entries, err := fs.ReadDir(migrationFS, migDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// collect .sql files and sort
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, fname := range files {
		// use filename (without extension) as migration version key
		version := strings.TrimSuffix(fname, path.Ext(fname))

		// check if already applied
		var count int
		row := d.QueryRow(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version)
		if row == nil {
			return fmt.Errorf("migration check query returned nil row for %s", version)
		}
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("scan migration applied count: %w", err)
		}
		if count > 0 {
			// already applied
			continue
		}

		// read and execute migration from embedded FS (use posix path.Join)
		p := path.Join(migDir, fname)
		b, err := fs.ReadFile(migrationFS, p)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fname, err)
		}
		if _, err := d.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("exec migration %s: %w", fname, err)
		}

		// record migration
		if _, err := d.Exec(ctx, `INSERT INTO schema_migrations (version, applied) VALUES (?, strftime('%s','now'))`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", fname, err)
		}
	}

	// optional seed files (attempt to read from provided seedFS; ignore not-found)
	schemaPath := path.Join("seed", "schema_v1.json")
	if b, err := fs.ReadFile(seedFS, schemaPath); err == nil {
		if _, err := d.Exec(ctx, `INSERT OR REPLACE INTO ai_schemas (version, description, schema_json, created, updated) VALUES ('v1', 'default v1 schema', ?, strftime('%s','now'), strftime('%s','now'))`, string(b)); err != nil {
			return fmt.Errorf("seed schema exec: %w", err)
		}
	}

	templatePath := path.Join("seed", "template_activity_v1.txt")
	if b, err := fs.ReadFile(seedFS, templatePath); err == nil {
		if _, err := d.Exec(ctx, `INSERT OR REPLACE INTO ai_templates (name, version, template_text, schema_version, metadata, created, updated) VALUES ('activity', 'v1', ?, ?, ?, strftime('%s','now'), strftime('%s','now'))`, string(b), "v1", `{"owner":"system","description":"default activity template"}`); err != nil {
			return fmt.Errorf("seed template exec: %w", err)
		}
	}

	return nil
}
