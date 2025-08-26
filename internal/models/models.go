package models

import (
	"encoding/json"
	"time"
)

type Engineer struct {
	ID           int64  `json:"id" db:"id"`
	Name         string `json:"name" db:"name" validate:"required"`
	Email        string `json:"email" db:"email" validate:"required,email"`
	Updated      int64  `json:"updated" db:"updated"`
	PasswordHash string `json:"password_hash,omitempty" db:"password_hash"`
}

type Profile struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Bio        string `json:"bio,omitempty" db:"bio"`
	Updated    int64  `json:"updated" db:"updated"`
}

type Activity struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Activity   string `json:"activity" db:"activity"`
	Created    int64  `json:"created" db:"created"`
}

type Question struct {
	ID         int64  `json:"id" db:"id"`
	EngineerID int64  `json:"engineer_id" db:"engineer_id"`
	Question   string `json:"question" db:"question"`
	Answered   *int64 `json:"answered,omitempty" db:"answered"`
	Created    int64  `json:"created" db:"created"`
}

type Job struct {
	ID      int64  `json:"id" db:"id"`
	Status  string `json:"status" db:"status"`
	Created int64  `json:"created" db:"created"`
}

type Schema struct {
	ID          int64  `json:"id" db:"id"`
	Version     string `json:"version" db:"version"`
	Description string `json:"description,omitempty" db:"description"`
	SchemaJSON  string `json:"schema_json" db:"schema_json"`
	Created     int64  `json:"created" db:"created"`
	Updated     int64  `json:"updated" db:"updated"`
}

type Template struct {
	ID          int64   `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Version     string  `json:"version" db:"version"`
	TemplateTxt string  `json:"template_text" db:"template_text"`
	SchemaVer   *string `json:"schema_version,omitempty" db:"schema_version"`
	Metadata    *string `json:"metadata,omitempty" db:"metadata"`
	Created     int64   `json:"created" db:"created"`
	Updated     int64   `json:"updated" db:"updated"`
}

type ContextHistory struct {
	ID            int64   `json:"id" db:"id"`
	EngineerID    int64   `json:"engineer_id" db:"engineer_id"`
	ContextJSON   string  `json:"context_json" db:"context_json"`
	ChangesJSON   *string `json:"changes_json,omitempty" db:"changes_json"`
	ConflictsJSON *string `json:"conflicts_json,omitempty" db:"conflicts_json"`
	AppliedBy     string  `json:"applied_by" db:"applied_by"`
	Created       int64   `json:"created" db:"created"`
	Version       int64   `json:"version" db:"version"`
}

type BackgroundJob struct {
	ID          int64           `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	Priority    int             `json:"priority"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	NextTryAt   *time.Time      `json:"next_try_at,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
	Created     time.Time       `json:"created"`
	Updated     time.Time       `json:"updated"`
}
