package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/garnizeh/rag/internal/models"
)

// Handler is the function that processes a job
type Handler func(ctx context.Context, j *models.BackgroundJob) error

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
