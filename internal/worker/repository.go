package worker

import (
	"context"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// updateWebhook is a struct representing all the necessary fields that may be
// used to update a webhook following a forwarding attempt.
type updateWebhook struct {
	id             uuid.UUID
	nextRetryAt    *time.Time
	deliveryStatus model.DeliveryStatus
	responseCode   *int
	lastError      *string
}

// WorkerKind enumerates the different types of worker processing strategies.
type WorkerKind int

const (
	// WorkerKindQueue processes newly ingested webhooks in 'queued' status.
	WorkerKindQueue WorkerKind = iota
	// WorkerKindRetry processes webhooks that previously failed and are due for retry.
	WorkerKindRetry
	// WorkerKindReplay processes webhooks that have been manually requested for replay.
	WorkerKindReplay
)

// WorkerRepository defines the contract for webhook event persistence
// operations required by background workers.
type WorkerRepository interface {
	// GetEvent queries the database for the next event to be processed.
	GetEvent(ctx context.Context) (*model.Webhook, error)
	// GetDestinationURL queries the database for the destination_url of
	// the provider with the given ID.
	GetDestinationURL(ctx context.Context, provID uuid.UUID) (string, error)
	// UpdateEvent applies the provided updates to the webhook event.
	UpdateEvent(ctx context.Context, updates updateWebhook) (*model.Webhook, error)
}

// WorkerRepo provides database operations for worker event processing.
// A single type handles queue, retry, and replay strategies via its
// WorkerKind field, avoiding duplicate code across previously separate
// repository types.
type WorkerRepo struct {
	db   *pgxpool.Pool
	kind WorkerKind
}

// NewWorkerRepo returns a WorkerRepo configured for the given processing kind.
func NewWorkerRepo(db *pgxpool.Pool, kind WorkerKind) WorkerRepository {
	return &WorkerRepo{
		db:   db,
		kind: kind,
	}
}

// GetEvent safely queries the database for the next event matching the
// worker's processing strategy. It uses SELECT FOR UPDATE SKIP LOCKED
// to prevent duplicate processing across concurrent workers.
func (r *WorkerRepo) GetEvent(ctx context.Context) (*model.Webhook, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var query string
	var args []any

	switch r.kind {
	case WorkerKindQueue:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'processing'
		WHERE id = (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'queued'
			ORDER BY received_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
	case WorkerKindRetry:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'retrying'
		WHERE id = (
			SELECT id FROM webhook_events
			WHERE
				(
					delivery_status = 'failed'
					AND next_retry_at <= NOW()
					AND retry_count < $1
				)
				OR (
					(delivery_status = 'processing' OR delivery_status = 'queued' OR delivery_status = 'replaying')
					AND updated_at < NOW() - INTERVAL '1 minute'
				)
			ORDER BY next_retry_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
		args = append(args, config.Envs.MaxRetries)
	case WorkerKindReplay:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'replaying'
		WHERE id = (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'replaying'
			ORDER BY received_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
	}

	var hook model.Webhook
	err = tx.QueryRow(ctx, query, args...).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt, &hook.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, tx.Commit(ctx)
}

// GetDestinationURL queries the database for the destination_url of the
// provider with the given ID.
func (r *WorkerRepo) GetDestinationURL(ctx context.Context, provID uuid.UUID) (string, error) {
	query := `SELECT destination_url FROM providers WHERE id = $1`
	var url string
	err := r.db.QueryRow(ctx, query, provID).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// UpdateEvent applies the provided updates to the webhook event.
// For retry workers it additionally increments the retry_count.
func (r *WorkerRepo) UpdateEvent(ctx context.Context, updates updateWebhook) (*model.Webhook, error) {
	var query string
	if r.kind == WorkerKindRetry {
		query = `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4,
			retry_count     = retry_count + 1
		WHERE id = $5
		RETURNING *`
	} else {
		query = `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4
		WHERE id = $5
		RETURNING *`
	}

	var hook model.Webhook
	err := r.db.QueryRow(ctx, query,
		updates.nextRetryAt, updates.deliveryStatus,
		updates.responseCode, updates.lastError, updates.id,
	).Scan(
		&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
		&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
		&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
		&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt, &hook.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &hook, nil
}
