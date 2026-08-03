package providers

import (
	"context"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderRepo provides database operations for managing providers.
type ProviderRepo struct {
	db *pgxpool.Pool
}

// NewProviderRepo returns a ProviderRepo backed by the provided connection pool.
func NewProviderRepo(db *pgxpool.Pool) *ProviderRepo {
	return &ProviderRepo{
		db: db,
	}
}

// GetAll retrieves all providers from the database.
func (r *ProviderRepo) getAll(ctx context.Context) ([]model.Provider, error) {
	query := `SELECT id, name, signing_secret, destination_url,
	                  max_retries, max_req_second,
	                  is_configured, created_at
	           FROM providers`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var provs []model.Provider
	for rows.Next() {
		var p model.Provider
		rowErr := rows.Scan(
			&p.ID, &p.Name, &p.SigningSecret,
			&p.DestinationURL, &p.MaxRetries, &p.MaxReqSecond,
			&p.IsConfigured, &p.CreatedAt,
		)
		if rowErr != nil {
			return provs, rowErr
		}
		provs = append(provs, p)
	}
	if err = rows.Err(); err != nil {
		return provs, err
	}
	return provs, nil
}

// Update modifies a provider's signing secret, destination URL,
// max_retries, and max_req_second, and sets is_configured flag to true
// if it isn't already, returning the updated Provider.
func (r *ProviderRepo) configure(
	ctx context.Context,
	id uuid.UUID,
	sec string,
	des string,
	maxRetry int,
	maxReqSec int,
) (model.Provider, error) {
	query := `
		UPDATE providers
		SET signing_secret = $1, destination_url = $2, is_configured = $3,
		    max_retries = $4, max_req_second = $5
		WHERE id = $6
		RETURNING id, name, signing_secret, destination_url,
		          max_retries, max_req_second, is_configured, created_at`
	var p model.Provider
	err := r.db.QueryRow(ctx, query, sec, des, true, maxRetry, maxReqSec, id).Scan(
		&p.ID,
		&p.Name,
		&p.SigningSecret,
		&p.DestinationURL,
		&p.MaxRetries,
		&p.MaxReqSecond,
		&p.IsConfigured,
		&p.CreatedAt,
	)
	if err != nil {
		return model.Provider{}, err
	}
	return p, nil
}
