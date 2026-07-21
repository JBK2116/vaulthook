package worker

import (
	"context"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env"); err != nil {
		panic(err)
	}
	config.Init()
	ctx := context.Background()
	db, err := config.NewPG(ctx)
	if err != nil {
		panic(err)
	}
	testDB = db.DB
	l, err := config.NewLogger()
	if err != nil {
		panic(err)
	}
	testLogger = l
	code := m.Run()
	os.Exit(code)
}

func beforeEachWorker(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset webhook_events: %v", err)
	}
	// Ensure a provider row exists for GetDestinationURL tests.
	_, err = testDB.Exec(context.Background(), `
		INSERT INTO providers (id, name, signing_secret, destination_url, is_configured)
		VALUES (gen_random_uuid(), 'Stripe', '', '', false)
		ON CONFLICT (name) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("failed to insert test provider: %v", err)
	}
}
