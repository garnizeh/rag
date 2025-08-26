-- Migration: add tables for background job processing

CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  payload TEXT,
  status TEXT NOT NULL DEFAULT 'queued',
  attempts INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 5,
  priority INTEGER NOT NULL DEFAULT 100,
  scheduled_at INTEGER NOT NULL,
  next_try_at INTEGER,
  last_error TEXT,
  created INTEGER NOT NULL,
  updated INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS dead_letter_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  payload TEXT,
  attempts INTEGER NOT NULL,
  last_error TEXT,
  failed_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_priority ON jobs(status, priority, next_try_at, scheduled_at);
