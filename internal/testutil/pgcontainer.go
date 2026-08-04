// Package testutil provides shared helpers for integration tests, including
// ephemeral PostgreSQL containers via testcontainers-go.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultDBName = "vaulthooktest"
	defaultUser   = "vaulthook"
	defaultPass   = "vaulthook"
	startTimeout  = 60 * time.Second
)

// NewTestDB starts an ephemeral Postgres 16-alpine container, runs the provided
// init scripts (typically migration .sql files), and returns a connection pool
// along with a cleanup function that terminates the container and closes the pool.
//
// Usage in TestMain:
//
//	pool, cleanup, err := testutil.NewTestDB(ctx,
//	    "../../migrations/00001_create_providers_table.sql",
//	    "../../migrations/00002_create_webhook_events_table.sql",
//	)
//	if err != nil { panic(err) }
//	defer cleanup()
//	testDB = pool
func NewTestDB(ctx context.Context, initScripts ...string) (*pgxpool.Pool, func(), error) {
	// Strip -- +goose Down sections so PostgreSQL doesn't execute the
	// destructive teardown SQL meant only for goose rollbacks.
	cleaned, cleanupFiles, err := stripGooseDown(initScripts...)
	if err != nil {
		return nil, nil, fmt.Errorf("testutil: strip goose: %w", err)
	}

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase(defaultDBName),
		postgres.WithUsername(defaultUser),
		postgres.WithPassword(defaultPass),
		postgres.WithInitScripts(cleaned...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(startTimeout),
		),
	)
	if err != nil {
		cleanupFiles()
		return nil, nil, fmt.Errorf("testutil: postgres container: %w", err)
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("testutil: connection string: %w", err)
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("testutil: parse config: %w", err)
	}
	cfg.MaxConns = 30

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("testutil: new pool: %w", err)
	}

	cleanup := func() {
		pool.Close()
		if err := ctr.Terminate(ctx); err != nil {
			fmt.Printf("testutil: terminate container: %v\n", err)
		}
		_ = testcontainers.TerminateContainer(ctr)
		cleanupFiles()
	}

	return pool, cleanup, nil
}

// stripGooseDown reads each file, truncates at the "-- +goose Down" marker,
// and writes the result to a temporary file. It returns the temp file paths
// and a cleanup function that removes them.
func stripGooseDown(paths ...string) ([]string, func(), error) {
	var tmpFiles []string
	cleanup := func() {
		for _, f := range tmpFiles {
			_ = os.Remove(f)
		}
	}

	cleaned := make([]string, len(paths))
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("read %s: %w", path, err)
		}

		content := string(data)
		if idx := strings.Index(content, "\n-- +goose Down"); idx >= 0 {
			content = content[:idx+1] // include the trailing newline
		}

		// Derive a prefix from the original filename so that PostgreSQL
		// init scripts execute in the correct order (alphabetical inside
		// /docker-entrypoint-initdb.d). Without this, random temp names
		// would cause out-of-order execution and container exit code 3.
		base := filepath.Base(path)
		prefix := base
		if before, _, ok := strings.Cut(base, "_"); ok {
			prefix = before
		}

		tmp, err := os.CreateTemp("", prefix+"-vaulthook-init-*.sql")
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("create temp: %w", err)
		}
		if _, err := tmp.WriteString(content); err != nil {
			tmp.Close()
			cleanup()
			return nil, nil, fmt.Errorf("write temp: %w", err)
		}
		tmp.Close()

		tmpFiles = append(tmpFiles, tmp.Name())
		cleaned[i] = tmp.Name()
	}

	return cleaned, cleanup, nil
}
