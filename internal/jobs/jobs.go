package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Job represents a background job
type Job struct {
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

// Handler is the function that processes a job
type Handler func(ctx context.Context, j *Job) error

// ErrMaxAttempts indicates the job reached max attempts
var ErrMaxAttempts = errors.New("max attempts reached")

// BackoffDuration returns exponential backoff duration for attempt n
func BackoffDuration(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	// simple exponential: base 2^attempt seconds, capped
	d := time.Duration(1<<uint(attempt)) * time.Second
	max := 5 * time.Minute
	if d > max {
		return max
	}
	return d
}

// small contract description
// inputs: job table rows, handlers map
// outputs: job status updates, dead-letter moves on permanent failure
// error modes: db errors, handler errors
