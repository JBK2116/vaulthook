package events

import (
	"context"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, cleanup, err := testutil.NewTestDB(ctx,
		"../../migrations/00001_create_providers_table.sql",
		"../../migrations/00002_create_webhook_events_table.sql",
		"../../migrations/00003_create_refresh_tokens_table.sql",
		"../../migrations/00004_insert_stripe_github_sns_default_config.sql",
	)
	if err != nil {
		panic(err)
	}
	defer cleanup()
	testDB = pool

	l := zerolog.Nop()
	testLogger = &l

	code := m.Run()
	os.Exit(code)
}

func beforeEachEvents(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset webhook_events: %v", err)
	}
}

func getProviderID(ctx context.Context, t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx, `SELECT id FROM providers LIMIT 1`).Scan(&id)
	if err != nil {
		t.Fatalf("no provider found: %v", err)
	}
	return id
}
