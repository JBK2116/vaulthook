package events

import (
	"context"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventRepo provides database operations for universally handling web events.
type EventRepo struct {
	db *pgxpool.Pool
}

// NewEventRepo returns a EventRepo backed by the provided connection pool.
func NewEventRepo(db *pgxpool.Pool) *EventRepo {
	return &EventRepo{
		db: db,
	}
}

// getAll retreives all webhook events from the database.
func (r *EventRepo) getAll(ctx context.Context, createdAt *time.Time) ([]model.Webhook, error) {
	var query string
	var rows pgx.Rows
	var err error
	if createdAt != nil {
		query = `
            SELECT * FROM webhook_events
            WHERE created_at < $1
            ORDER BY created_at DESC
            LIMIT 5`
		rows, err = r.db.Query(ctx, query, createdAt)
	} else {
		query = `
            SELECT * FROM webhook_events
            ORDER BY created_at DESC
            LIMIT 25`
		rows, err = r.db.Query(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		err = rows.Scan(
			&w.ID, &w.ProviderID, &w.Provider, &w.EventID,
			&w.EventType, &w.Headers, &w.Payload, &w.DeliveryStatus,
			&w.ForwardedTo, &w.ResponseCode, &w.RetryCount, &w.NextRetryAt,
			&w.LastError, &w.ReceivedAt, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, w)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return hooks, nil
}

// getStats retrieves the count statistics of the webhook processing events that took place in the last 7 days
func (r *EventRepo) getStats(ctx context.Context) (*model.Stats, error) {
	query := ` 
	SELECT 
		COUNT(*) FILTER (WHERE delivery_status = 'delivered') as delivered,
		COUNT(*) FILTER (WHERE delivery_status = 'failed') as failed,
		COUNT(*) FILTER (WHERE delivery_status = 'retrying') as retrying,
		COUNT(*) FILTER (WHERE delivery_status = 'queued') as queued
	FROM webhook_events
	WHERE created_at > NOW() - INTERVAL '7 days'

	`
	var stats model.Stats
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.Delivered,
		&stats.Failed,
		&stats.Retrying,
		&stats.Queued,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// replayEvent sets the delivery_status of the webhook with the provided ID to "queued",
// allowing it to be picked by queue workers to be replayed.
func (r *EventRepo) replayEvent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE webhook_events SET delivery_status = 'replaying' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}
