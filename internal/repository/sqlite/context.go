package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/garnizeh/rag/internal/models"
)

// UpsertEngineerContext inserts or updates the current context and returns the new version.
func (r *SQLiteRepo) UpsertEngineerContext(ctx context.Context, engineerID int64, contextJSON string, appliedBy string) (int64, error) {
	// attempt update first
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// check existing
	var curVersion sql.NullInt64
	var exists int
	row := tx.QueryRowContext(ctx, `SELECT version FROM engineer_contexts WHERE engineer_id = ?`, engineerID)
	if err := row.Scan(&curVersion); err != nil {
		if err == sql.ErrNoRows {
			exists = 0
		} else {
			return 0, fmt.Errorf("query existing context: %w", err)
		}
	} else {
		exists = 1
	}

	now := now()
	var newVersion int64 = 1
	if exists == 1 {
		newVersion = curVersion.Int64 + 1
		if _, err := tx.ExecContext(ctx, `UPDATE engineer_contexts SET context_json = ?, version = ?, updated = ? WHERE engineer_id = ?`, contextJSON, newVersion, now, engineerID); err != nil {
			return 0, fmt.Errorf("update context: %w", err)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineer_contexts (engineer_id, context_json, version, updated) VALUES (?, ?, ?, ?)`, engineerID, contextJSON, newVersion, now); err != nil {
			return 0, fmt.Errorf("insert context: %w", err)
		}
	}

	// create history entry
	if _, err := tx.ExecContext(ctx, `INSERT INTO engineer_context_history (engineer_id, context_json, applied_by, created, version) VALUES (?, ?, ?, ?, ?)`, engineerID, contextJSON, appliedBy, now, newVersion); err != nil {
		return 0, fmt.Errorf("create context history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return newVersion, nil
}

// GetEngineerContext returns the current context JSON and version, or empty string + 0 if none.
func (r *SQLiteRepo) GetEngineerContext(ctx context.Context, engineerID int64) (string, int64, error) {
	row := r.conn.QueryRow(ctx, `SELECT context_json, version FROM engineer_contexts WHERE engineer_id = ?`, engineerID)
	var ctxJSON sql.NullString
	var version sql.NullInt64
	if err := row.Scan(&ctxJSON, &version); err != nil {
		if err == sql.ErrNoRows {
			return "", 0, nil
		}
		return "", 0, err
	}
	return ctxJSON.String, version.Int64, nil
}

// CreateContextHistory creates an explicit history entry (useful when changes are not saved via Upsert).
func (r *SQLiteRepo) CreateContextHistory(ctx context.Context, engineerID int64, contextJSON string, changesJSON *string, conflictsJSON *string, appliedBy string, version int64) (int64, error) {
	res, err := r.conn.Exec(ctx, `INSERT INTO engineer_context_history (engineer_id, context_json, changes_json, conflicts_json, applied_by, created, version) VALUES (?, ?, ?, ?, ?, ?, ?)`, engineerID, contextJSON, changesJSON, conflictsJSON, appliedBy, now(), version)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListContextHistory returns history items for an engineer ordered by created desc.
func (r *SQLiteRepo) ListContextHistory(ctx context.Context, engineerID int64) ([]models.ContextHistory, error) {
	rows, err := r.conn.QueryRows(ctx, `SELECT id, engineer_id, context_json, changes_json, conflicts_json, applied_by, created, version FROM engineer_context_history WHERE engineer_id = ? ORDER BY created DESC`, engineerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ContextHistory
	for rows.Next() {
		var h models.ContextHistory
		var changes sql.NullString
		var conflicts sql.NullString
		if err := rows.Scan(&h.ID, &h.EngineerID, &h.ContextJSON, &changes, &conflicts, &h.AppliedBy, &h.Created, &h.Version); err != nil {
			return nil, err
		}
		if changes.Valid {
			s := changes.String
			h.ChangesJSON = &s
		}
		if conflicts.Valid {
			s := conflicts.String
			h.ConflictsJSON = &s
		}
		out = append(out, h)
	}

	return out, nil
}

// GetContextHistoryByID returns a single history entry by its id for an engineer.
func (r *SQLiteRepo) GetContextHistoryByID(ctx context.Context, engineerID int64, historyID int64) (*models.ContextHistory, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, engineer_id, context_json, changes_json, conflicts_json, applied_by, created, version FROM engineer_context_history WHERE engineer_id = ? AND id = ?`, engineerID, historyID)
	var h models.ContextHistory
	var changes sql.NullString
	var conflicts sql.NullString
	if err := row.Scan(&h.ID, &h.EngineerID, &h.ContextJSON, &changes, &conflicts, &h.AppliedBy, &h.Created, &h.Version); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if changes.Valid {
		s := changes.String
		h.ChangesJSON = &s
	}
	if conflicts.Valid {
		s := conflicts.String
		h.ConflictsJSON = &s
	}
	return &h, nil
}
