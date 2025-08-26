package sqlite

import (
	"context"
	"fmt"

	"github.com/garnizeh/rag/internal/models"
)

func (r *SQLiteRepo) CreateJob(ctx context.Context, j *models.Job) (int64, error) {
	if j == nil {
		return 0, fmt.Errorf("job is nil")
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO processing_jobs (status, created) VALUES (?, ?)`, j.Status, now())
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SQLiteRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.conn.Exec(ctx, `UPDATE processing_jobs SET status = ? WHERE id = ?`, status, id)
	return err
}
