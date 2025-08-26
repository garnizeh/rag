package jobs_test

import (
	"context"
	"testing"
	"time"

	"log/slog"

	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/internal/jobs"
	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/internal/repository/sqlite"
)

func TestEnqueueAndProcess(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	// use shared in-memory DB so multiple connections see the same schema
	d, err := db.New(ctx, "file::memory:?cache=shared", logger)
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	defer d.Close()

	// run migrations - create jobs tables
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT NOT NULL, payload TEXT, status TEXT NOT NULL DEFAULT 'queued', attempts INTEGER NOT NULL DEFAULT 0, max_attempts INTEGER NOT NULL DEFAULT 5, priority INTEGER NOT NULL DEFAULT 100, scheduled_at INTEGER NOT NULL DEFAULT (strftime('%s','now')), next_try_at INTEGER, last_error TEXT, created INTEGER NOT NULL DEFAULT (strftime('%s','now')), updated INTEGER NOT NULL DEFAULT (strftime('%s','now')))`); err != nil {
		t.Fatalf("create jobs table: %v", err)
	}
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS dead_letter_jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, job_id INTEGER NOT NULL, type TEXT NOT NULL, payload TEXT, attempts INTEGER NOT NULL, last_error TEXT, failed_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))) `); err != nil {
		t.Fatalf("create dlq table: %v", err)
	}

	repo := sqlite.New(d, logger)
	handled := make(chan struct{}, 1)
	handlers := map[string]jobs.Handler{
		"test": func(ctx context.Context, j *models.BackgroundJob) error {
			handled <- struct{}{}
			return nil
		},
	}
	pool := jobs.NewWorkerPool(repo, handlers, logger, 1)
	pool.Start(ctx)
	defer pool.Stop()

	if _, err := pool.Enqueue(ctx, "test", map[string]string{"foo": "bar"}, 10, 3); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	select {
	case <-handled:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatalf("handler was not called")
	}
}
