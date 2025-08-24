package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garnizeh/rag/pkg/models"
)

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
