package events

import (
	"context"
	"fmt"
	"strings"
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

// InsertWebhook saves a new webhook event to the database and returns the
// stored record with all database-generated fields populated.
func (r *EventRepo) InsertWebhook(ctx context.Context, p model.CreateWebhookParams) (model.Webhook, error) {
	query := `
	INSERT INTO webhook_events (provider_id, provider, event_id, event_type, headers, payload, forwarded_to, received_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	RETURNING id, provider_id, provider, event_id, event_type, headers, payload,
	          delivery_status, forwarded_to, response_code, retry_count,
	          next_retry_at, last_error, received_at, created_at, updated_at
	`

	var w model.Webhook
	err := r.db.QueryRow(ctx, query,
		p.ProviderID, p.Provider, p.EventID, p.EventType,
		p.Headers, p.Payload, p.ForwardedTo, p.ReceivedAt,
	).Scan(
		&w.ID, &w.ProviderID, &w.Provider, &w.EventID, &w.EventType,
		&w.Headers, &w.Payload, &w.DeliveryStatus, &w.ForwardedTo,
		&w.ResponseCode, &w.RetryCount, &w.NextRetryAt, &w.LastError,
		&w.ReceivedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return model.Webhook{}, err
	}
	return w, nil
}

// replayEvent sets the delivery_status of the webhook with the provided ID to "replaying",
// allowing it to be picked by replay workers to be replayed.
func (r *EventRepo) replayEvent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE webhook_events SET delivery_status = 'replaying' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

// lookup retrieves webhooks matching either the provided webhook ID or event ID.
// At least one of the two IDs must be non-nil and non-empty.
func (r *EventRepo) lookup(ctx context.Context, opts model.LookupOpts) ([]model.Webhook, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	if opts.WebhookID != nil && *opts.WebhookID != "" {
		conditions = append(conditions, fmt.Sprintf("id::text = $%d", argIdx))
		args = append(args, *opts.WebhookID)
		argIdx++
	}
	if opts.EventID != nil && *opts.EventID != "" {
		conditions = append(conditions, fmt.Sprintf("event_id = $%d", argIdx))
		args = append(args, *opts.EventID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT * FROM webhook_events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " OR "),
		argIdx, argIdx+1,
	)
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(
			&w.ID, &w.ProviderID, &w.Provider, &w.EventID,
			&w.EventType, &w.Headers, &w.Payload, &w.DeliveryStatus,
			&w.ForwardedTo, &w.ResponseCode, &w.RetryCount, &w.NextRetryAt,
			&w.LastError, &w.ReceivedAt, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		hooks = append(hooks, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hooks, nil
}

// filter retrieves webhooks matching the criteria in opts. Only fields that are
// explicitly set (non-zero / non-empty) become filter conditions.
func (r *EventRepo) filter(ctx context.Context, opts model.FilterOpts) ([]model.Webhook, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	if len(opts.Providers) > 0 {
		conditions = append(conditions, fmt.Sprintf("provider = ANY($%d)", argIdx))
		args = append(args, opts.Providers)
		argIdx++
	}
	if opts.EventType != nil && *opts.EventType != "" {
		conditions = append(conditions, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, *opts.EventType)
		argIdx++
	}
	if len(opts.DeliveryStatuses) > 0 {
		conditions = append(conditions, fmt.Sprintf("delivery_status = ANY($%d)", argIdx))
		args = append(args, opts.DeliveryStatuses)
		argIdx++
	}
	if opts.ResponseCode != nil {
		conditions = append(conditions, fmt.Sprintf("response_code = $%d", argIdx))
		args = append(args, *opts.ResponseCode)
		argIdx++
	}
	if opts.FromTime != nil && opts.ToTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *opts.FromTime)
		argIdx++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *opts.ToTime)
		argIdx++
	}
	if opts.PayloadSearch != nil && *opts.PayloadSearch != "" {
		conditions = append(conditions, fmt.Sprintf("payload::text ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, *opts.PayloadSearch)
		argIdx++
	}
	if opts.HasRetries {
		conditions = append(conditions, "retry_count > 0")
	}
	if opts.HasError {
		conditions = append(conditions, "last_error IS NOT NULL")
	}

	query := fmt.Sprintf(`
		SELECT * FROM webhook_events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "),
		argIdx, argIdx+1,
	)
	args = append(args, opts.Limit, opts.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(
			&w.ID, &w.ProviderID, &w.Provider, &w.EventID,
			&w.EventType, &w.Headers, &w.Payload, &w.DeliveryStatus,
			&w.ForwardedTo, &w.ResponseCode, &w.RetryCount, &w.NextRetryAt,
			&w.LastError, &w.ReceivedAt, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		hooks = append(hooks, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hooks, nil
}
