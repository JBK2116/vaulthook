package stripe

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StripeRepo provides database operations for managing stripe webhooks
type StripeRepo struct {
	db *pgxpool.Pool
}

// NewStripeRepo returns a StripeRepo backed by the provided connection pool
func NewStripeRepo(db *pgxpool.Pool) *StripeRepo {
	return &StripeRepo{
		db: db,
	}
}

// getSigningKey returns the signing_key associated from the provider with the matching name
func (r *StripeRepo) getSigningKey(ctx context.Context, provider string) (string, error) {
	query := `SELECT signing_secret FROM providers WHERE name = $1`
	var signingKey string
	err := r.db.QueryRow(ctx, query, provider).Scan(&signingKey)
	if err != nil {
		return "", err
	}
	return signingKey, nil
}

// insertWebhook saves a webhook to the database and returns the stored record.
// Fields not provided (e.g., ID, status, timestamps) are set by the database.
func (r *StripeRepo) insertWebhook(ctx context.Context, p providers.CreateWebhookParams) (providers.Webhook, error) {
	query := `
	INSERT INTO webhook_events (provider_id, provider, event_id, event_type, headers, payload, forwarded_to, received_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	RETURNING id, provider_id, provider, event_id, event_type, headers, payload,
	          delivery_status, forwarded_to, response_code, retry_count,
	          next_retry_at, last_error, received_at, created_at
	`

	var w providers.Webhook
	err := r.db.QueryRow(ctx, query,
		p.ProviderID, p.Provider, p.EventID, p.EventType,
		p.Headers, p.Payload, p.ForwardedTo, p.ReceivedAt,
	).Scan(
		&w.ID, &w.ProviderID, &w.Provider, &w.EventID, &w.EventType,
		&w.Headers, &w.Payload, &w.DeliveryStatus, &w.ForwardedTo,
		&w.ResponseCode, &w.RetryCount, &w.NextRetryAt, &w.LastError,
		&w.ReceivedAt, &w.CreatedAt,
	)
	if err != nil {
		return providers.Webhook{}, err
	}

	return w, nil
}
