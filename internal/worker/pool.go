package worker

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// WorkerPool is a struct that orchestrates all webhook workers.
//
// There must be only one worker pool present throughout the entire application.
type WorkerPool struct {
	signal  chan struct{}
	workers []*Worker
}

// NewWorkerPool returns a WorkerPool backed by the provided configuration
func NewWorkerPool(ctx context.Context, svc *events.EventService, logger *zerolog.Logger, db *pgxpool.Pool) *WorkerPool {
	signal := make(chan struct{}, 1)
	workers := make([]*Worker, config.Envs.TotalQueueWorkers+config.Envs.TotalRetryWorkers)
	// initialize the repo for the workers
	queueRepo := NewQueueWorkerRepo(db)
	retryRepo := NewRetryWorkerRepo(db)
	// initialize the QueueWorkers
	for i := 0; i < config.Envs.TotalQueueWorkers; i++ {
		workers[i] = newWorker(svc, queueRepo, logger)
	}
	// initialize the RetryWorkers
	for i := config.Envs.TotalQueueWorkers; i < len(workers); i++ {
		workers[i] = newWorker(svc, retryRepo, logger)
	}
	// initialize the worker pool
	pool := &WorkerPool{signal: signal, workers: workers}
	pool.start(ctx)
	return pool
}

// start kicks off the worker runtime cycle for each worker in the pool.
func (p *WorkerPool) start(ctx context.Context) {
	for _, w := range p.workers {
		go w.start(ctx, p.signal)
	}
}

// Notify alerts all workers in the pool that one or more webhooks need to be processed.
func (p *WorkerPool) Notify() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}
