package sqlite

import (
	"context"
	"fmt"

	"github.com/garnizeh/rag/pkg/models"
)

func (r *SQLiteRepo) CreateActivity(ctx context.Context, a *models.Activity) (int64, error) {
	if a == nil {
		return 0, fmt.Errorf("activity is nil")
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO raw_activities (engineer_id, activity, created) VALUES (?, ?, ?)`, a.EngineerID, a.Activity, a.Created)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SQLiteRepo) ListByEngineer(ctx context.Context, engineerID int64, limit, offset int) ([]models.Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.conn.QueryRows(ctx, `SELECT id, engineer_id, activity, created FROM raw_activities WHERE engineer_id = ? ORDER BY created DESC LIMIT ? OFFSET ?`, engineerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Activity
	for rows.Next() {
		var a models.Activity
		if err := rows.Scan(&a.ID, &a.EngineerID, &a.Activity, &a.Created); err != nil {
			return nil, err
		}

		out = append(out, a)
	}

	return out, nil
}

func (r *SQLiteRepo) CountActivitiesByEngineer(ctx context.Context, engineerID int64) (int64, error) {
	row := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM raw_activities WHERE engineer_id = ?`, engineerID)
	var cnt int64
	if err := row.Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}
