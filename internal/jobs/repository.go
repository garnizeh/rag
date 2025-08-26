package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/garnizeh/rag/internal/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

// Enqueue inserts a job into the jobs table and returns the new ID
func (r *Repository) Enqueue(ctx context.Context, j *Job) (int64, error) {
	payload := string(j.Payload)
	if j.MaxAttempts == 0 {
		j.MaxAttempts = 5
	}
	now := time.Now().UTC().UnixMilli()
	q := `INSERT INTO jobs(type, payload, status, attempts, max_attempts, priority, scheduled_at, created, updated) VALUES(?,?,?,?,?,?,?,?,?)`
	res, err := r.db.Exec(ctx, q, j.Type, payload, "queued", j.Attempts, j.MaxAttempts, j.Priority, j.ScheduledAt.UTC().Unix(), now, now)
	if err != nil {
		return 0, fmt.Errorf("enqueue failed: %w", err)
	}
	return res.LastInsertId()
}

// FetchNext fetches the next available job respecting priority and schedule
func (r *Repository) FetchNext(ctx context.Context) (*Job, error) {
	// select job where status queued or retry and next_try_at <= now and scheduled_at <= now
	q := `SELECT id, type, payload, status, attempts, max_attempts, priority, scheduled_at, next_try_at, last_error, created, updated FROM jobs WHERE (status = 'queued' OR status = 'retry') AND (next_try_at IS NULL OR next_try_at <= ?) AND scheduled_at <= ? ORDER BY priority ASC, scheduled_at ASC LIMIT 1`
	now := time.Now().UTC().Unix()
	row := r.db.QueryRow(ctx, q, now, now)
	var (
		id          int64
		typ         string
		payload     sql.NullString
		status      string
		attempts    int
		maxAttempts int
		priority    int
		scheduledAt int64
		nextTry     sql.NullInt64
		lastError   sql.NullString
		created     int64
		updated     int64
	)
	if err := row.Scan(&id, &typ, &payload, &status, &attempts, &maxAttempts, &priority, &scheduledAt, &nextTry, &lastError, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch next job: %w", err)
	}
	j := &Job{
		ID:          id,
		Type:        typ,
		Status:      status,
		Attempts:    attempts,
		MaxAttempts: maxAttempts,
		Priority:    priority,
		ScheduledAt: time.Unix(scheduledAt, 0),
		Created:     time.Unix(created, 0),
		Updated:     time.Unix(updated, 0),
	}
	if payload.Valid {
		j.Payload = json.RawMessage(payload.String)
	}
	if nextTry.Valid {
		t := time.Unix(nextTry.Int64, 0)
		j.NextTryAt = &t
	}
	if lastError.Valid {
		j.LastError = lastError.String
	}
	return j, nil
}

// UpdateJob updates attempts, status, next_try_at, last_error
func (r *Repository) UpdateJob(ctx context.Context, j *Job) error {
	var nextTry interface{}
	if j.NextTryAt != nil {
		nextTry = j.NextTryAt.Unix()
	} else {
		nextTry = nil
	}
	q := `UPDATE jobs SET status = ?, attempts = ?, next_try_at = ?, last_error = ?, updated = ? WHERE id = ?`
	_, err := r.db.Exec(ctx, q, j.Status, j.Attempts, nextTry, j.LastError, time.Now().UTC().Unix(), j.ID)
	return err
}

// MoveToDeadLetter moves a job to dead_letter_jobs and deletes the original
func (r *Repository) MoveToDeadLetter(ctx context.Context, j *Job) error {
	tx, err := r.db.GetConn().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	payload := string(j.Payload)
	insert := `INSERT INTO dead_letter_jobs(job_id, type, payload, attempts, last_error, failed_at) VALUES(?,?,?,?,?,?)`
	if _, err := tx.ExecContext(ctx, insert, j.ID, j.Type, payload, j.Attempts, j.LastError, time.Now().UTC().Unix()); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, j.ID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
