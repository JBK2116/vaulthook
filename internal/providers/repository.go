package providers

import (
	"context"

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
func (r *ProviderRepo) getAll(ctx context.Context) ([]Provider, error) {
	query := `SELECT * FROM providers`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var provs []Provider
	for rows.Next() {
		var p Provider
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.SigningSecret,
			&p.DestinationURL,
			&p.IsConfigured,
			&p.CreatedAt,
		)
		if err != nil {
			return provs, err
		}
		provs = append(provs, p)
	}
	if err = rows.Err(); err != nil {
		return provs, err
	}
	return provs, nil
}

// Update modifies a provider's signing secret and destination URL, and
// sets is_configured flag to true if it isn't already, returning the updated Provider.
func (r *ProviderRepo) configure(
	ctx context.Context,
	id uuid.UUID,
	sec string,
	des string,
) (Provider, error) {
	query := `
		UPDATE providers
		SET signing_secret = $1, destination_url = $2, is_configured = $3
		WHERE id = $4
		RETURNING id, name, signing_secret, destination_url, is_configured, created_at`
	var p Provider
	err := r.db.QueryRow(ctx, query, sec, des, true, id).Scan(
		&p.ID,
		&p.Name,
		&p.SigningSecret,
		&p.DestinationURL,
		&p.IsConfigured,
		&p.CreatedAt,
	)
	if err != nil {
		return Provider{}, err
	}
	return p, nil
}

// GetProviderRouting retrieves the routing configuration for a given provider.
// It looks up the provider by name and returns its unique identifier along with
// the destination URL where incoming webhooks should be forwarded.
//
// Returns an error if the provider cannot be found or if the query fails.
func (r *ProviderRepo) GetProviderRouting(ctx context.Context, name string) (ProviderRouting, error) {
	query := `SELECT id, destination_url FROM providers WHERE name = $1`
	var pr ProviderRouting
	err := r.db.QueryRow(ctx, query, name).Scan(&pr.ID, &pr.ForwardedTo)
	if err != nil {
		return ProviderRouting{}, err
	}
	return pr, nil
}
