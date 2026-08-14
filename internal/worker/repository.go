package worker

import (
	"context"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// updateWebhook is a struct representing all the necessary fields that may be
// used to update a webhook following a forwarding attempt.
type updateWebhook struct {
	id             uuid.UUID
	provName       model.ProviderName
	nextRetryAt    *time.Time
	deliveryStatus model.DeliveryStatus
	responseCode   *int
	lastError      *string
	forwardedTo    *string
}

// WorkerKind enumerates the different types of worker processing strategies.
type WorkerKind int

// sensible batch defaults override in app.go
var (
	QueueWorkerBatch = 50
	RetryWorkerBatch = 25
)

const ReplayWorkerBatch = 1

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
	// GetEvents queries the database for the next batch of events to be processed.
	GetEvents(ctx context.Context) ([]model.Webhook, error)
	// UpdateEvents applies the provided updates to the webhook events in batch.
	UpdateEvents(ctx context.Context, updates []updateWebhook) ([]model.Webhook, error)
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

// GetEvents safely queries the database for the next event matching the
// worker's processing strategy. It uses SELECT FOR UPDATE SKIP LOCKED
// to prevent duplicate processing across concurrent workers.
func (r *WorkerRepo) GetEvents(ctx context.Context) ([]model.Webhook, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var query string
	var limit int

	switch r.kind {
	case WorkerKindQueue:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'processing'
		WHERE id IN (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'queued'
			ORDER BY received_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
		limit = QueueWorkerBatch
	case WorkerKindRetry:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'retrying'
		WHERE id IN (
			SELECT id FROM webhook_events
			WHERE
				(
					delivery_status = 'failed'
					AND next_retry_at <= NOW()
					AND retry_count < (
						SELECT max_retries FROM providers WHERE id = webhook_events.provider_id
					)
				)
				OR (
					(delivery_status = 'processing' OR delivery_status = 'queued' OR delivery_status = 'replaying')
					AND updated_at < NOW() - INTERVAL '1 minute'
				)
			ORDER BY next_retry_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
		limit = RetryWorkerBatch
	case WorkerKindReplay:
		query = `
		UPDATE webhook_events
		SET delivery_status = 'replaying'
		WHERE id IN (
			SELECT id FROM webhook_events
			WHERE delivery_status = 'replaying'
			ORDER BY received_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		) RETURNING *`
		limit = ReplayWorkerBatch
	}
	var hooks []model.Webhook
	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var hook model.Webhook
		if err := rows.Scan(&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
			&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
			&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
			&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt, &hook.UpdatedAt,
		); err != nil {
			return hooks, err
		}
		hooks = append(hooks, hook)
	}
	if rErr := rows.Err(); rErr != nil {
		return nil, rErr
	}
	return hooks, tx.Commit(ctx)
}

// UpdateEvents applies the provided updates to the webhook events in batch.
// For retry workers it additionally increments the retry_count.
// Uses pgx.Batch to send all updates in a single network round-trip.
func (r *WorkerRepo) UpdateEvents(ctx context.Context, updates []updateWebhook) ([]model.Webhook, error) {
	if len(updates) == 0 {
		return []model.Webhook{}, nil
	}

	var query string
	if r.kind == WorkerKindRetry {
		query = `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4,
			forwarded_to    = $5,
			retry_count     = retry_count + 1
		WHERE id = $6
		RETURNING *`
	} else {
		query = `
		UPDATE webhook_events
		SET
			next_retry_at   = $1,
			delivery_status = $2,
			response_code   = $3,
			last_error      = $4,
			forwarded_to    = $5
		WHERE id = $6
		RETURNING *`
	}

	batch := &pgx.Batch{}
	for _, u := range updates {
		batch.Queue(query,
			u.nextRetryAt, u.deliveryStatus,
			u.responseCode, u.lastError, u.forwardedTo, u.id,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	hooks := make([]model.Webhook, 0, len(updates))
	for range updates {
		var hook model.Webhook
		err := br.QueryRow().Scan(
			&hook.ID, &hook.ProviderID, &hook.Provider, &hook.EventID,
			&hook.EventType, &hook.Headers, &hook.Payload, &hook.DeliveryStatus,
			&hook.ForwardedTo, &hook.ResponseCode, &hook.RetryCount, &hook.NextRetryAt,
			&hook.LastError, &hook.ReceivedAt, &hook.CreatedAt, &hook.UpdatedAt,
		)
		if err != nil {
			return hooks, err
		}
		hooks = append(hooks, hook)
	}
	return hooks, nil
}
