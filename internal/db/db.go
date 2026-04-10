// Package db is responsible for database related configuration logic
//
// This package handles connecting the database,  managing pooling and related logic.
// It exposes a database object that is passed throughout the application.
package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// postgres manages a psql connection pool used to query and execute statements in the database
type postgres struct {
	db *pgxpool.Pool
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

// NewPG uses `sync.Once` to create a new pgxpool database connection.
// If successful a postgres struct is returned, else an error is returned.
func NewPG(ctx context.Context, connString string) (*postgres, error) {
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

// Ping checks if the postgres db is properly connected to the database
func (pg *postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

// Close is responsible for closing all connections in the pool and rejecting all future calls permanently
func (pg *postgres) Close() {
	pg.db.Close()
}
