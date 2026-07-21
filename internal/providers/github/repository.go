package github

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GitRepo provides database operations for managing github webhooks
type GitRepo struct {
	db *pgxpool.Pool
}

// NewGitRepo returns a GitRepo backed by the provided connection pool
func NewGitRepo(db *pgxpool.Pool) *GitRepo {
	return &GitRepo{
		db: db,
	}
}

// getSigningKey returns the signing key associated with the provider with the matching name
func (r *GitRepo) getSigningKey(ctx context.Context, provider string) (string, error) {
	query := `SELECT signing_secret FROM providers WHERE name = $1`
	var key string
	err := r.db.QueryRow(ctx, query, provider).Scan(&key)
	if err != nil {
		return "", err
	}
	return key, nil
}

// insertWebhook saves a webhook to the database and returns the stored record.
// Fields not provided (e.g., ID, status, timestamps) are set by the database.
func (r *GitRepo) insertWebhook(ctx context.Context, p model.CreateWebhookParams) (model.Webhook, error) {
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
