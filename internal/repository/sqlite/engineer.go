package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garnizeh/rag/pkg/models"
)

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
