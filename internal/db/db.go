// Package db manages the application's database connection pool.
//
// It handles pool initialization and exposes a postgres instance
// that is passed throughout the application.
package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgres holds the pgxpool connection pool used to query and execute
// statements against the PostgreSQL database.
type postgres struct {
	DB *pgxpool.Pool
}

var (
	pgInstance *postgres // singleton postgres instance.
	pgOnce     sync.Once // guards pool initialization.
)

// NewPG constructs and returns a singleton postgres instance backed by a
// pgxpool connection pool. The pool is initialized exactly once via sync.Once.
// Subsequent calls return the existing instance.
//
// The connection string is assembled from config.Envs. If the pool cannot
// be created, an error is returned and pgInstance remains nil.
func NewPG(ctx context.Context) (*postgres, error) {
	connString := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		config.Envs.DB_TYPE,
		config.Envs.DB_USER,
		config.Envs.DB_PASSWORD,
		config.Envs.DB_HOST,
		config.Envs.DB_PORT,
		config.Envs.DB_NAME,
	)
	var err error
	pgOnce.Do(func() {
		db, pgErr := pgxpool.New(ctx, connString)
		if pgErr != nil {
			err = fmt.Errorf("unable to create connection pool: %w", pgErr)
			return
		}
		pgInstance = &postgres{db}
	})
	return pgInstance, err
}

// Ping verifies the database connection is alive.
func (pg *postgres) Ping(ctx context.Context) error {
	return pg.DB.Ping(ctx)
}

// Close terminates all connections in the pool. It is permanent;
// the pool cannot be reused after Close is called.
func (pg *postgres) Close() {
	pg.DB.Close()
}
