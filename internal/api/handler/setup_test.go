package handler

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger
var testRepo *auth.RefreshTokenRepo
var testService *auth.AuthService
var testHandler *authHandler

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../.env.test"); err != nil {
		panic(err)
	}
	config.Init()
	ctx := context.Background()
	db, err := db.NewPG(ctx)
	if err != nil {
		panic(err)
	}
	testDB = db.DB
	l, err := logger.NewLogger()
	if err != nil {
		panic(err)
	}
	testLogger = l
	r := auth.NewRefreshTokenRepo(testDB)
	testRepo = r
	s := auth.NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, r, l)
	testService = s
	h := NewAuthHandler(l, s)
	testHandler = h
	code := m.Run()
	os.Exit(code)
}

// beforeEach acts as a setup function responsible for running code before each test begins
func beforeEach(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events, providers RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to reset tables: %v", err)
	}

}

// afterEach acts as a teardown function responsible for running code after each test ends
func afterEach(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events, providers RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to cleanup tables: %v", err)
	}
}

// getValidLoginCredentials returns an array of bytes containing the json equivalent of a valid login request body
func getValidLoginCredentials() []byte {
	loginCreds := loginRequestBody{Email: config.Envs.UserEmail, Password: config.Envs.UserPassword}
	body, _ := json.Marshal(&loginCreds)
	return body
}
