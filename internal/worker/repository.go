package worker

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/google/uuid"
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

// GetEvent safely queries the database for the next event in the queue.
func (r *QueueWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	// id is the tie breaker to selecting events from the db, when they were added sequentially quick
	query := `
		UPDATE webhook_events
		SET delivery_status = 'processing'
		WHERE id = (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'queued'
			ORDER BY received_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`
	var hook providers.Webhook
	err = tx.QueryRow(ctx, query).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, tx.Commit(ctx)
}

// GetDestinationURL queries the database for the destination_url of the provider with the provided ID
func (r *QueueWorkerRepo) GetDestinationURL(ctx context.Context, provID uuid.UUID) (string, error) {
	query := `SELECT destination_url FROM providers WHERE id = $1`
	var url string
	err := r.db.QueryRow(ctx, query, provID).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// UpdateEvent updates the necessary values of the provided webhook event.
//
// Updated values include
//
//	next_retry_at
//	delivery_status
//	response_code
//	last_error
func (r *QueueWorkerRepo) UpdateEvent(ctx context.Context, updates updateWebhook) (*providers.Webhook, error) {
	query := `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4
		WHERE id = $5
		RETURNING *`

	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query, updates.nextRetryAt, updates.deliveryStatus, updates.responseCode, updates.lastError, updates.id).Scan(
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

// GetEvent safely queries the database for the next event in the queue.
func (r *RetryWorkerRepo) GetEvent(ctx context.Context) (*providers.Webhook, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	// id is the tie breaker here if next_retry_at is equal between events
	query := `
		UPDATE webhook_events
		SET delivery_status = 'processing'
		WHERE id = (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'retrying'
			AND next_retry_at <= NOW()
			AND retry_count < $1
			ORDER BY next_retry_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`
	var hook providers.Webhook
	err = tx.QueryRow(ctx, query, config.Envs.MaxRetries).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, tx.Commit(ctx)
}

// GetDestinationURL queries the database for the destination_url of the provider with the provided ID
func (r *RetryWorkerRepo) GetDestinationURL(ctx context.Context, provID uuid.UUID) (string, error) {
	query := `SELECT destination_url FROM providers WHERE id = $1`
	var url string
	err := r.db.QueryRow(ctx, query, provID).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// UpdateEvent updates the necessary values of the provided webhook event.
//
// Updated values include
//
//	next_retry_at
//	delivery_status
//	response_code
//	last_error
func (r *RetryWorkerRepo) UpdateEvent(ctx context.Context, updates updateWebhook) (*providers.Webhook, error) {
	query := `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4
		WHERE id = $5
		RETURNING *`
	var hook providers.Webhook
	err := r.db.QueryRow(ctx, query, updates.nextRetryAt, updates.deliveryStatus, updates.responseCode, updates.lastError, updates.id).Scan(
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
