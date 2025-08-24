-- Migration: add tables for AI schemas and templates

CREATE TABLE IF NOT EXISTS ai_schemas (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  version TEXT NOT NULL UNIQUE,
  description TEXT,
  schema_json TEXT NOT NULL,
  created INTEGER NOT NULL DEFAULT (strftime('%s','now')),
  updated INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE TABLE IF NOT EXISTS ai_templates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  template_text TEXT NOT NULL,
  schema_version TEXT,
  metadata TEXT,
  created INTEGER NOT NULL DEFAULT (strftime('%s','now')),
  updated INTEGER NOT NULL DEFAULT (strftime('%s','now')),
  UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_templates_version ON ai_templates(version);
CREATE INDEX IF NOT EXISTS idx_schemas_version ON ai_schemas(version);
