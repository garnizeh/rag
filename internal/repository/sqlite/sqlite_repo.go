package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/garnizeh/rag/internal/db"
	"github.com/garnizeh/rag/pkg/models"
	"github.com/garnizeh/rag/pkg/repository"
)

// SQLiteRepo implements repository interfaces using the internal DB wrapper.
type SQLiteRepo struct {
	conn *db.DB
}

// Ensure SQLiteRepo implements the public interfaces.
var _ repository.EngineerRepo = (*SQLiteRepo)(nil)
var _ repository.ProfileRepo = (*SQLiteRepo)(nil)
var _ repository.ActivityRepo = (*SQLiteRepo)(nil)
var _ repository.QuestionRepo = (*SQLiteRepo)(nil)
var _ repository.JobRepo = (*SQLiteRepo)(nil)

func New(conn *db.DB) *SQLiteRepo {
	return &SQLiteRepo{conn: conn}
}

// Engineer methods
func (r *SQLiteRepo) CreateEngineer(ctx context.Context, e *models.Engineer) (int64, error) {
	if e == nil {
		return 0, fmt.Errorf("engineer is nil")
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO engineers (name, email, updated, password_hash) VALUES (?, ?, ?, ?)`, e.Name, e.Email, now(), e.PasswordHash)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SQLiteRepo) GetByID(ctx context.Context, id int64) (*models.Engineer, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, name, email, updated, password_hash FROM engineers WHERE id = ?`, id)
	var e models.Engineer
	var pw sql.NullString
	if err := row.Scan(&e.ID, &e.Name, &e.Email, &e.Updated, &pw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	if pw.Valid {
		e.PasswordHash = pw.String
	}

	return &e, nil
}

func (r *SQLiteRepo) GetByEmail(ctx context.Context, email string) (*models.Engineer, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, name, email, updated, password_hash FROM engineers WHERE email = ?`, email)
	var e models.Engineer
	var pw sql.NullString
	if err := row.Scan(&e.ID, &e.Name, &e.Email, &e.Updated, &pw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	if pw.Valid {
		e.PasswordHash = pw.String
	}

	return &e, nil
}

func (r *SQLiteRepo) UpdateEngineer(ctx context.Context, e *models.Engineer) error {
	if e == nil {
		return fmt.Errorf("engineer is nil")
	}

	_, err := r.conn.Exec(ctx, `UPDATE engineers SET name = ?, email = ?, updated = ?, password_hash = ? WHERE id = ?`, e.Name, e.Email, now(), e.PasswordHash, e.ID)
	return err
}

func (r *SQLiteRepo) DeleteEngineer(ctx context.Context, id int64) error {
	_, err := r.conn.Exec(ctx, `DELETE FROM engineers WHERE id = ?`, id)
	return err
}

// Profile methods
func (r *SQLiteRepo) CreateProfile(ctx context.Context, p *models.Profile) (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("profile is nil")
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO engineer_profiles (engineer_id, bio, updated) VALUES (?, ?, ?)`, p.EngineerID, p.Bio, now())
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SQLiteRepo) GetByEngineerID(ctx context.Context, engineerID int64) (*models.Profile, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, engineer_id, bio, updated FROM engineer_profiles WHERE engineer_id = ?`, engineerID)
	var p models.Profile
	if err := row.Scan(&p.ID, &p.EngineerID, &p.Bio, &p.Updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &p, nil
}

func (r *SQLiteRepo) UpdateProfile(ctx context.Context, p *models.Profile) error {
	if p == nil {
		return fmt.Errorf("profile is nil")
	}

	_, err := r.conn.Exec(ctx, `UPDATE engineer_profiles SET bio = ?, updated = ? WHERE id = ?`, p.Bio, now(), p.ID)
	return err
}

func (r *SQLiteRepo) DeleteProfile(ctx context.Context, id int64) error {
	_, err := r.conn.Exec(ctx, `DELETE FROM engineer_profiles WHERE id = ?`, id)
	return err
}

// Activity methods
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

	rows, err := r.conn.GetConn().QueryContext(ctx, `SELECT id, engineer_id, activity, created FROM raw_activities WHERE engineer_id = ? ORDER BY created DESC LIMIT ? OFFSET ?`, engineerID, limit, offset)
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

// CountActivitiesByEngineer returns the total number of activities for an engineer.
func (r *SQLiteRepo) CountActivitiesByEngineer(ctx context.Context, engineerID int64) (int64, error) {
	row := r.conn.GetConn().QueryRowContext(ctx, `SELECT COUNT(*) FROM raw_activities WHERE engineer_id = ?`, engineerID)
	var cnt int64
	if err := row.Scan(&cnt); err != nil {
		return 0, err
	}
	return cnt, nil
}

// Question methods
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

// Job methods
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

func now() int64 {
	return time.Now().UTC().UnixMilli()
}
