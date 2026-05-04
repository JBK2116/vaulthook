package worker

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/config"
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

// GetEvent queries the database for the next event in the queue.
func (r *QueueWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	query := `
    UPDATE webhook_events
    SET delivery_status = 'processing'
    WHERE id = (
        SELECT id FROM webhook_events
        WHERE delivery_status = 'queued'
        ORDER BY received_at ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    )
    RETURNING *`
	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, nil
}

// UpdateEventStatus updates the necessary values of the provided webhook event.
func (r *QueueWorkerRepo) UpdateEventStatus(ctx context.Context, evt UpdateEvent) (*providers.Webhook, error) {
	query := `
		UPDATE webhook_events
		SET
			delivery_status = $1,
			response_code   = $2,
			last_error      = $3
		WHERE id = $4
		RETURNING *`

	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query, evt.DeliveryStatus, evt.ResponseCode, evt.LastError, evt.ID).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, nil
}

// GetEvent queries the database for the next event in the queue.
//
// GetEvent updates next_retry_at and retry_count automatically and automatically
func (r *RetryWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	query := `
    UPDATE webhook_events
    SET 
		next_retry_at = NOW() + ($1 * INTERVAL '1 second'),
		retry_count = retry_count + 1
    WHERE id = (
        SELECT id FROM webhook_events
        WHERE delivery_status = 'retrying'
        AND next_retry_at <= NOW()
		AND retry_count < $2
        ORDER BY next_retry_at ASC, id ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    )
    RETURNING *`
	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query, config.Envs.RetryIntervalSeconds, config.Envs.MaxRetries).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, nil
}

// UpdateEventStatus updates the necessary values of the provided webhook event.
func (r *RetryWorkerRepo) UpdateEventStatus(ctx context.Context, evt UpdateEvent) (*providers.Webhook, error) {
	query := `
		UPDATE webhook_events
		SET
			delivery_status = $1,
			response_code   = $2,
			last_error      = $3
		WHERE id = $4
		RETURNING *`

	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query, evt.DeliveryStatus, evt.ResponseCode, evt.LastError, evt.ID).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, nil
}
