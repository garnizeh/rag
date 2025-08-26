package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garnizeh/rag/internal/models"
)

func (r *SQLiteRepo) CreateQuestion(ctx context.Context, q *models.Question) (int64, error) {
	if q == nil {
		return 0, fmt.Errorf("question is nil")
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO ai_questions (engineer_id, question, created) VALUES (?, ?, ?)`, q.EngineerID, q.Question, now())
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SQLiteRepo) ListUnansweredByEngineer(ctx context.Context, engineerID int64) ([]models.Question, error) {
	rows, err := r.conn.QueryRows(ctx, `SELECT id, engineer_id, question, answered, created FROM ai_questions WHERE engineer_id = ? AND answered IS NULL ORDER BY created DESC`, engineerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Question
	for rows.Next() {
		var q models.Question
		var answered sql.NullInt64
		if err := rows.Scan(&q.ID, &q.EngineerID, &q.Question, &answered, &q.Created); err != nil {
			return nil, err
		}

		if answered.Valid {
			v := answered.Int64
			q.Answered = &v
		}
		out = append(out, q)
	}

	return out, nil
}
