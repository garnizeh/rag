-- Migration: add engineer context storage and history for AI-driven updates

CREATE TABLE IF NOT EXISTS engineer_contexts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  engineer_id INTEGER NOT NULL UNIQUE,
  context_json TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  updated INTEGER NOT NULL,
  FOREIGN KEY(engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS engineer_context_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  engineer_id INTEGER NOT NULL,
  context_json TEXT NOT NULL,
  changes_json TEXT, -- array of ChangeRecord
  conflicts_json TEXT, -- array of conflict strings
  applied_by TEXT, -- who/what applied the change (ai, user, rollback)
  created INTEGER NOT NULL,
  version INTEGER NOT NULL,
  FOREIGN KEY(engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_contexts_engineer ON engineer_contexts(engineer_id);
CREATE INDEX IF NOT EXISTS idx_context_history_engineer_created ON engineer_context_history(engineer_id, created);
