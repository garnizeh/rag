package jobs_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"log/slog"

	"github.com/garnizeh/rag/internal/ai"
	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/internal/jobs"
	"github.com/garnizeh/rag/internal/models"
	"github.com/garnizeh/rag/internal/repository/sqlite"
	"github.com/garnizeh/rag/pkg/repository"
)

// Test that an ai.process_response job invokes ProcessAIResponse and persists context
func TestAIProcessJobIntegration(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	d, err := db.New(ctx, "file::memory:?cache=shared", logger)
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	defer d.Close()

	// create minimal tables: jobs and contexts + history
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, type TEXT NOT NULL, payload TEXT, status TEXT NOT NULL DEFAULT 'queued', attempts INTEGER NOT NULL DEFAULT 0, max_attempts INTEGER NOT NULL DEFAULT 5, priority INTEGER NOT NULL DEFAULT 100, scheduled_at INTEGER NOT NULL, next_try_at INTEGER, last_error TEXT, created INTEGER NOT NULL, updated INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create jobs table: %v", err)
	}
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS engineer_contexts (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER NOT NULL UNIQUE, context_json TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1, updated INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create contexts table: %v", err)
	}
	if _, err := d.Exec(ctx, `CREATE TABLE IF NOT EXISTS engineer_context_history (id INTEGER PRIMARY KEY AUTOINCREMENT, engineer_id INTEGER NOT NULL, context_json TEXT NOT NULL, changes_json TEXT, conflicts_json TEXT, applied_by TEXT, created INTEGER NOT NULL, version INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create contexts history table: %v", err)
	}

	sqliteRepo := sqlite.New(d, logger)

	repo := sqlite.New(d, logger)

	// ensure ai processor logger is set to a working logger to avoid nil writer panic
	ai.SetProcessorLogger(logger)

	// reuse sqliteRepo for domain repositories used by ProcessAIResponse
	// build handler map
	handlers := map[string]jobs.Handler{
		"ai.process_response": func(ctx context.Context, j *models.BackgroundJob) error {
			// payload expected: {"engineer_id":123, "response": <AIResponse JSON>}
			var pl struct {
				EngineerID int64           `json:"engineer_id"`
				Response   json.RawMessage `json:"response"`
			}
			if err := json.Unmarshal(j.Payload, &pl); err != nil {
				return err
			}
			var resp ai.AIResponse
			if err := json.Unmarshal(pl.Response, &resp); err != nil {
				return err
			}
			// construct repository.Repository mapping to sqliteRepo implementations
			r := &repository.Repository{Context: sqliteRepo, Question: sqliteRepo}
			_, err := ai.ProcessAIResponse(ctx, r, pl.EngineerID, &resp)
			return err
		},
	}

	pool := jobs.NewWorkerPool(repo, handlers, logger, 1)
	pool.Start(ctx)
	defer pool.Stop()

	// enqueue job with a simple AIResponse payload
	resp := ai.AIResponse{Summary: "Test summary", ContextUpdate: true}
	bresp, _ := json.Marshal(resp)
	payload := map[string]any{"engineer_id": 123, "response": json.RawMessage(bresp)}
	if _, err := pool.Enqueue(ctx, "ai.process_response", payload, 10, 3); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	// wait for processing to persist context
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ctxJSON, version, err := sqliteRepo.GetEngineerContext(ctx, 123)
		if err == nil && version > 0 && ctxJSON != "" {
			// success
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("ai processing job did not persist context in time")
}
