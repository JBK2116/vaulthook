package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger
var testRepo *RefreshTokenRepo
var testService *AuthService

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env.test"); err != nil {
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
	r := NewRefreshTokenRepo(testDB)
	testRepo = r
	s := NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, r, l)
	testService = s
	code := m.Run()
	os.Exit(code)
}

// beforeEach acts as a setup function responsible for running code before each test begins
func beforeEach(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset tables: %v", err)
	}

}

// afterEach acts as a teardown function responsible for running code after each test ends
func afterEach(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset tables: %v", err)
	}
}

// CreateRefreshToken generates a valid refresh token, saves it to the database and returns it to the user
func createRefreshToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Duration(config.Envs.RefreshTokenTTL) * time.Hour)
	token, err := testService.GenerateRefreshToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tokenStruct, err := testRepo.Create(ctx, token, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return tokenStruct.Token
}

// createExpiredRefreshToken generates an expired refresh token, saves it to the database and returns it to the user
func createExpiredRefreshToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Minute * -1)
	token, err := testService.GenerateRefreshToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tokenStruct, err := testRepo.Create(ctx, token, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return tokenStruct.Token
}

// createExpiredAccessToken generates an expired access token and returns it to the user
func createExpiredAccessToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Minute * -1)
	token, err := testService.GenerateAccessToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
