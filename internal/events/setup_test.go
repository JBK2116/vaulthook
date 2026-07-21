package events

import (
	"context"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/google/uuid"
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
