package sqlite

import (
	"context"
	"database/sql"

	"github.com/garnizeh/rag/internal/models"
)

func (r *SQLiteRepo) CreateTemplate(ctx context.Context, name, version, templateText string, schemaVersion *string, metadata *string) (int64, error) {
	var schemaVer any
	var meta any
	if schemaVersion != nil {
		schemaVer = *schemaVersion
	} else {
		schemaVer = nil
	}
	if metadata != nil {
		meta = *metadata
	} else {
		meta = nil
	}

	res, err := r.conn.Exec(ctx, `INSERT INTO ai_templates (name, version, template_text, schema_version, metadata, created, updated) VALUES (?, ?, ?, ?, ?, strftime('%s','now'), strftime('%s','now')) ON CONFLICT(name, version) DO UPDATE SET template_text=excluded.template_text, schema_version=excluded.schema_version, metadata=excluded.metadata, updated=strftime('%s','now')`, name, version, templateText, schemaVer, meta)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *SQLiteRepo) GetTemplate(ctx context.Context, name, version string) (*models.Template, error) {
	row := r.conn.QueryRow(ctx, `SELECT id, name, version, template_text, schema_version, metadata, created, updated FROM ai_templates WHERE name = ? AND version = ?`, name, version)
	var t models.Template
	if err := row.Scan(&t.ID, &t.Name, &t.Version, &t.TemplateTxt, &t.SchemaVer, &t.Metadata, &t.Created, &t.Updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *SQLiteRepo) ListTemplates(ctx context.Context) ([]models.Template, error) {
	rows, err := r.conn.QueryRows(ctx, `SELECT id, name, version, template_text, schema_version, metadata, created, updated FROM ai_templates ORDER BY name, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Template
	for rows.Next() {
		var t models.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Version, &t.TemplateTxt, &t.SchemaVer, &t.Metadata, &t.Created, &t.Updated); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *SQLiteRepo) DeleteTemplate(ctx context.Context, name, version string) error {
	_, err := r.conn.Exec(ctx, `DELETE FROM ai_templates WHERE name = ? AND version = ?`, name, version)
	return err
}
