package worker

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewQueueWorkerRepo returns a WorkerRepo backed by the provided database connection.
func NewQueueWorkerRepo(db *pgxpool.Pool) WorkerRepository {
	return &QueueWorkerRepo{
		db: db,
	}
}

// NewRetryWorkerRepo returns a WorkerRepository backed by the provided database connection.
func NewRetryWorkerRepo(db *pgxpool.Pool) WorkerRepository {
	return &RetryWorkerRepo{
		db: db,
	}
}

func (r *QueueWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	// TODO: Implement This
}

func (r *QueueWorkerRepo) UpdateEventStatus(ctx context.Context, event *providers.Webhook) (*providers.Webhook, error) {
	// TODO: Implement This
}

func (r *RetryWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	// TODO: Implement This
}

func (r *RetryWorkerRepo) UpdateEventStatus(ctx context.Context, event *providers.Webhook) (*providers.Webhook, error) {
	// TODO: Implement This
}
