package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

type WorkerPool struct {
	repo        *Repository
	handlers    map[string]Handler
	logger      *slog.Logger
	workerCount int
	stop        chan struct{}
	wg          sync.WaitGroup
}

func NewWorkerPool(repo *Repository, handlers map[string]Handler, logger *slog.Logger, workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 4
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkerPool{repo: repo, handlers: handlers, logger: logger, workerCount: workerCount, stop: make(chan struct{})}
}

// Start launches the worker goroutines
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Stop signals workers to stop and waits for them
func (p *WorkerPool) Stop() {
	close(p.stop)
	p.wg.Wait()
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			p.logger.Info("worker stopping", "id", id)
			return
		case <-ctx.Done():
			p.logger.Info("context canceled, worker exiting", "id", id)
			return
		default:
			job, err := p.repo.FetchNext(ctx)
			if err != nil {
				p.logger.Error("fetch job", "err", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if job == nil {
				// nothing to do
				time.Sleep(500 * time.Millisecond)
				continue
			}
			h, ok := p.handlers[job.Type]
			if !ok {
				job.Status = "failed"
				job.LastError = "no handler"
				_ = p.repo.MoveToDeadLetter(ctx, job)
				continue
			}
			// run handler with context and cancellation
			err = h(ctx, job)
			if err == nil {
				job.Status = "done"
				_ = p.repo.UpdateJob(ctx, job)
				continue
			}
			// handler returned error
			job.Attempts++
			job.LastError = err.Error()
			if job.Attempts >= job.MaxAttempts {
				// move to dead letter
				job.Status = "failed"
				if mvErr := p.repo.MoveToDeadLetter(ctx, job); mvErr != nil {
					p.logger.Error("move to dead letter", "err", mvErr)
				}
				continue
			}
			// schedule retry with backoff
			backoff := BackoffDuration(job.Attempts)
			t := time.Now().Add(backoff)
			job.NextTryAt = &t
			job.Status = "retry"
			if upErr := p.repo.UpdateJob(ctx, job); upErr != nil {
				p.logger.Error("update job for retry", "err", upErr)
			}
		}
	}
}

// Enqueue convenience helper that creates a job and persists it
func (p *WorkerPool) Enqueue(ctx context.Context, typ string, payload any, priority int, maxAttempts int) (int64, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	j := &Job{Type: typ, Payload: b, Priority: priority, MaxAttempts: maxAttempts, ScheduledAt: time.Now()}
	return p.repo.Enqueue(ctx, j)
}
