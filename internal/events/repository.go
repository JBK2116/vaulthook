package events

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/model"
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
func (r *EventRepo) getAll(ctx context.Context) ([]model.Webhook, error) {
	query := `SELECT * FROM webhook_events`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		err := rows.Scan(
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
