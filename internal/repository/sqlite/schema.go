package sqlite

import (
	"context"
	"database/sql"

	"github.com/garnizeh/rag/pkg/models"
)

// CreateSchema inserts or updates a schema by version.
func (r *SQLiteRepo) CreateSchema(ctx context.Context, version, description, schemaJSON string) (int64, error) {
	// upsert using INSERT OR REPLACE pattern
	res, err := r.conn.Exec(ctx, `INSERT INTO ai_schemas (version, description, schema_json, created, updated) VALUES (?, ?, ?, strftime('%s','now'), strftime('%s','now')) ON CONFLICT(version) DO UPDATE SET description=excluded.description, schema_json=excluded.schema_json, updated=strftime('%s','now')`, version, description, schemaJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *SQLiteRepo) GetSchemaByVersion(ctx context.Context, version string) (*models.Schema, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, version, description, schema_json, created, updated FROM ai_schemas WHERE version = ?`, version)
	var s models.Schema
	if err := row.Scan(&s.ID, &s.Version, &s.Description, &s.SchemaJSON, &s.Created, &s.Updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SQLiteRepo) ListSchemas(ctx context.Context) ([]models.Schema, error) {
	rows, err := r.conn.QueryRows(ctx, `SELECT id, version, description, schema_json, created, updated FROM ai_schemas ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Schema
	for rows.Next() {
		var s models.Schema
		if err := rows.Scan(&s.ID, &s.Version, &s.Description, &s.SchemaJSON, &s.Created, &s.Updated); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *SQLiteRepo) DeleteSchema(ctx context.Context, version string) error {
	_, err := r.conn.Exec(ctx, `DELETE FROM ai_schemas WHERE version = ?`, version)
	return err
}
