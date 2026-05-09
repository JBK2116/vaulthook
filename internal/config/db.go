package config

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// postgres holds the pgxpool connection pool used to query and execute
// statements against the PostgreSQL database.
type postgres struct {
	DB *pgxpool.Pool
}

var (
	pgInstance *postgres       // singleton postgres instance.
	pgOnce     sync.Once       // guards pool initialization.
	PgErr      *pgconn.PgError // represents all possible pgx errors
)

// NewPG constructs and returns a singleton postgres instance backed by a
// pgxpool connection pool. The pool is initialized exactly once via sync.Once.
// Subsequent calls return the existing instance.
//
// The connection string is assembled from config.Envs. If the pool cannot
// be created, an error is returned and pgInstance remains nil.
func NewPG(ctx context.Context) (*postgres, error) {
	connString := fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		Envs.DBType,
		Envs.DBUser,
		Envs.DBPassword,
		Envs.DBHost,
		Envs.DBPort,
		Envs.DBName,
	)
	var err error
	pgOnce.Do(func() {
		cfg, cfgErr := pgxpool.ParseConfig(connString)
		if cfgErr != nil {
			err = fmt.Errorf("unable to parse connection string: %w", err)
			return
		}
		cfg.MaxConns = 50
		db, pgErr := pgxpool.NewWithConfig(ctx, cfg)
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
