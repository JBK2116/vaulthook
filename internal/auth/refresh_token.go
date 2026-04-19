package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID    uuid.UUID
	Token string
}

type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{
		db: db,
	}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token string, exp time.Time, iat time.Time) (*RefreshToken, error) {
	query := `INSERT INTO refresh_tokens (token, expires_at, created_at) VALUES ($1, $2, $3) RETURNING id`
	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, token, exp, iat).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &RefreshToken{
		ID:    id,
		Token: token,
	}, nil
}

func (r *RefreshTokenRepo) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	if _, err := r.db.Exec(ctx, query, token); err != nil {
		return err
	}
	return nil
}

func (r *RefreshTokenRepo) Exists(ctx context.Context, token string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE token = $1)`
	if err := r.db.QueryRow(ctx, query, token).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
