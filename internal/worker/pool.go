package worker

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/JBK2116/vaulthook/internal/events"
)

// Default worker counts, tuned in app.go based on computed resources before the pool is constructed.
var (
	//nolint:gochecknoglobals // Tunable worker-count config, assigned from app.go at startup; intentional.
	TotalQueueWorkers = 8
	//nolint:gochecknoglobals // Tunable worker-count config, assigned from app.go at startup; intentional.
	TotalRetryWorkers = 4
)

// Pool is a struct that orchestrates all webhook workers.
//
// There must be only one worker pool present throughout the entire application.
type Pool struct {
	signal       chan struct{}
	queueWorkers []*Worker
	retryWorkers []*Worker
	replayWorker *Worker
	cleanup      *cleanupWorker
}

// NewWorkerPool returns a WorkerPool backed by the provided configuration.
func NewWorkerPool(ctx context.Context, svc *events.Service, logger *zerolog.Logger, db *pgxpool.Pool) *Pool {
	signal := make(chan struct{}, TotalQueueWorkers)
	queueWorkers := make([]*Worker, TotalQueueWorkers)
	retryWorkers := make([]*Worker, TotalRetryWorkers)

	// initialize repos for each worker strategy
	queueRepo := NewWorkerRepo(db, WorkerKindQueue)
	retryRepo := NewWorkerRepo(db, WorkerKindRetry)
	replayRepo := NewWorkerRepo(db, WorkerKindReplay)

	// initialize the QueueWorkers
	for i := range queueWorkers {
		queueWorkers[i] = newWorker(svc, queueRepo, logger)
	}
	// initialize the RetryWorkers
	for i := range retryWorkers {
		retryWorkers[i] = newWorker(svc, retryRepo, logger)
	}
	// initialize the replayWorker
	replayWorker := newWorker(svc, replayRepo, logger)
	// initialize the cleanupWorker
	cleanupW := NewCleanupWorker(logger, db)
	// initialize the worker pool
	pool := &Pool{
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
func (p *Pool) start(ctx context.Context) {
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
func (p *Pool) Notify() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}
