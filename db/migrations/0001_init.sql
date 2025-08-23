-- Initial migration: create all required tables

CREATE TABLE engineers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    updated INTEGER NOT NULL
);

CREATE TABLE engineer_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engineer_id INTEGER NOT NULL,
    bio TEXT,
    updated INTEGER NOT NULL,
    FOREIGN KEY(engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE TABLE raw_activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engineer_id INTEGER NOT NULL,
    activity TEXT NOT NULL,
    created INTEGER NOT NULL,
    FOREIGN KEY(engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE TABLE ai_questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    engineer_id INTEGER NOT NULL,
    question TEXT NOT NULL,
    answered INTEGER,
    created INTEGER NOT NULL,
    FOREIGN KEY(engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);

CREATE TABLE processing_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    status TEXT NOT NULL,
    created INTEGER NOT NULL
);

-- Indexes for performance
CREATE INDEX idx_activities_engineer_created ON raw_activities(engineer_id, created);
CREATE INDEX idx_questions_engineer_answered ON ai_questions(engineer_id, answered);
CREATE INDEX idx_jobs_status_created ON processing_jobs(status, created);
CREATE INDEX idx_engineers_updated ON engineers(updated);
CREATE INDEX idx_profiles_updated ON engineer_profiles(updated);
