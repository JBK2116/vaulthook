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
	signal       chan struct{}
	queueWorkers []*Worker
	retryWorkers []*Worker
	replayWorker *Worker
	cleanup      *cleanupWorker
}

// NewWorkerPool returns a WorkerPool backed by the provided configuration
func NewWorkerPool(ctx context.Context, svc *events.EventService, logger *zerolog.Logger, db *pgxpool.Pool) *WorkerPool {
	signal := make(chan struct{}, config.Envs.TotalQueueWorkers)
	queueWorkers := make([]*Worker, config.Envs.TotalQueueWorkers)
	retryWorkers := make([]*Worker, config.Envs.TotalRetryWorkers)
	// initialize the repo for the workers
	queueRepo := NewQueueWorkerRepo(db)
	retryRepo := NewRetryWorkerRepo(db)
	replayRepo := NewReplayWorkerRepo(db)
	// initialize the QueueWorkers
	for i := range len(queueWorkers) {
		queueWorkers[i] = newWorker(svc, queueRepo, logger)
	}
	// initialize the RetryWorkers
	for i := range len(retryWorkers) {
		retryWorkers[i] = newWorker(svc, retryRepo, logger)
	}
	// initialize the replayWorker
	replayWorker := newWorker(svc, replayRepo, logger)
	// initialize the cleanupWorker
	cleanupW := NewCleanupWorker(logger, db)
	// initialize the worker pool
	pool := &WorkerPool{
		signal:       signal,
		queueWorkers: queueWorkers,
		retryWorkers: retryWorkers,
		replayWorker: replayWorker,
		cleanup:      cleanupW,
	}
	pool.start(ctx)
	return pool
}

// start kicks off the worker runtime cycle for each worker in the pool.
func (p *WorkerPool) start(ctx context.Context) {
	for _, w := range p.queueWorkers {
		go w.start(ctx, p.signal)
	}
	for _, w := range p.retryWorkers {
		go w.startRetry(ctx)
	}
	go p.replayWorker.startReplay(ctx)
	go p.cleanup.startCleanup(ctx)
}

// Notify alerts all workers in the pool that one or more webhooks need to be processed.
func (p *WorkerPool) Notify() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}
