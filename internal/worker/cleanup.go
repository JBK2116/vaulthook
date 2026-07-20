package worker

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const (
	cleanupInterval = time.Hour * 1
	maxRows         = 500_000
	retentionDays   = 7
)

// CleanupWorker struct is responsible for cleaning up old and
// excessive webhooks from the database.
type cleanupWorker struct {
	logger *zerolog.Logger
	db     *pgxpool.Pool
}

// NewCleanupWorker returns a cleanupWorker backed by the provided database connection and logger.
func NewCleanupWorker(logger *zerolog.Logger, db *pgxpool.Pool) *cleanupWorker {
	return &cleanupWorker{
		logger: logger,
		db:     db,
	}
}

// startCleanup kicks off the cleanup worker to begin
// periodically cleaning up rows from the database.
func (w *cleanupWorker) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runCleanup(ctx)
		}
	}
}

// runCleanup starts the cycle for cleaning up webhooks from the database.
func (w *cleanupWorker) runCleanup(ctx context.Context) {
	// delete events older than 7 days
	age, err := w.db.Exec(ctx, `
		DELETE FROM webhook_events
		WHERE received_at < NOW() - INTERVAL '7 days'
	`)
	if err != nil {
		w.logger.Error().Err(err).Msg("[Cleanup] age purge error")
		return
	}
	if age.RowsAffected() > 0 {
		w.logger.Info().Int64("rows_removed", age.RowsAffected()).Int("retention_days", retentionDays).Msg("[Cleanup] age purge complete")
	}
	// if still over limit, evict oldest first
	var count int
	err = w.db.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events`).Scan(&count)
	if err != nil {
		w.logger.Error().Err(err).Msg("[Cleanup] row count error")
		return
	}
	if count <= maxRows {
		return
	}
	excess := count - maxRows
	evict, err := w.db.Exec(ctx, `
		DELETE FROM webhook_events
		WHERE id IN (
			SELECT id FROM webhook_events
			ORDER BY received_at ASC
			LIMIT $1
		)
	`, excess)
	if err != nil {
		w.logger.Error().Err(err).Msg("[Cleanup] eviction error")
		return
	}
	w.logger.Info().Int64("rows_removed", evict.RowsAffected()).Int("max_rows", maxRows).Msg("[cleanup] eviction complete")
}
