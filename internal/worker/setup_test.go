package worker

import (
	"context"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/cache"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger

func TestMain(m *testing.M) {
	// Load .env for Redis config.
	if err := godotenv.Load("../../.env"); err != nil {
		panic(err)
	}
	config.Init()

	ctx := context.Background()
	if err := cache.InitRedisCache(ctx); err != nil {
		panic(err)
	}
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

func beforeEachWorker(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset webhook_events: %v", err)
	}
	// Ensure a provider row exists with clean defaults for each test.
	_, err = testDB.Exec(context.Background(), `
		INSERT INTO providers (id, name, signing_secret, destination_url, max_retries, max_req_second, is_configured)
		VALUES (gen_random_uuid(), 'Stripe', '', '', 5, 10, false)
		ON CONFLICT (name) DO UPDATE SET destination_url = '', max_req_second = 10
	`)
	if err != nil {
		t.Fatalf("failed to insert test provider: %v", err)
	}
	// Populate the in-memory provider cache used by forwardEvent.
	provRepo := providers.NewProviderRepo(testDB)
	if err := providers.InitProviderCache(context.Background(), provRepo); err != nil {
		t.Fatalf("failed to init provider cache: %v", err)
	}
	// Initialise per-provider rate-limiter buckets.
	InitRateLimiter()
}
