package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	crypto "github.com/JBK2116/vaulthook/internal/crpyto"
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/logger"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger

// EVENT HANDLING
var eventRepo *events.EventRepo
var eventService *events.EventService

// AUTH
var testAuthRepo *auth.RefreshTokenRepo
var testAuthService *auth.AuthService
var testAuthHandler *authHandler

// PROVIDERS
var testProviderRepo *providers.ProviderRepo
var testProviderService *providers.ProviderService
var testProviderHandler *providerHandler

// STRIPE
var stripeRepo *stripe.StripeRepo
var stripeService *stripe.StripeService
var stripeHandle *stripeHandler

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
	// configure the event variables
	eventR := events.NewEventRepo(db.DB)
	eventRepo = eventR
	eventS := events.NewEventService(l, eventR)
	eventService = eventS
	// configure the auth variables
	authR := auth.NewRefreshTokenRepo(testDB)
	testAuthRepo = authR
	authS := auth.NewAuthService(config.Envs.JWTSecret, config.Envs.AccessTokenTTL, config.Envs.RefreshTokenTTL, authR, l)
	testAuthService = authS
	authH := NewAuthHandler(l, authS)
	testAuthHandler = authH
	// configure the provider variables
	providerR := providers.NewProviderRepo(db.DB)
	testProviderRepo = providerR
	providerS := providers.NewProviderService(providerR)
	testProviderService = providerS
	providerH := NewProviderHandler(l, providerS)
	testProviderHandler = providerH
	// configure the stripe variables
	stripeR := stripe.NewStripeRepo(db.DB)
	stripeRepo = stripeR
	stripeS := stripe.NewStripeService(l, stripeR, providerR)
	stripeService = stripeS
	stripeH := NewStripeHandler(l, stripeS, eventS)
	stripeHandle = stripeH
	// run the code
	code := m.Run()
	os.Exit(code)
}

// beforeEach acts as a setup function responsible for running code before each test begins
func beforeEach(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to reset tables: %v", err)
	}
	_, err = testDB.Exec(context.Background(), "UPDATE providers SET destination_url = $1, signing_secret = $2 WHERE name = $3", "", "", providers.Stripe)
	if err != nil {
		t.Fatalf("failed to reset tables :%v", err)
	}
}

// afterEach acts as a teardown function responsible for running code after each test ends
func afterEach(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(context.Background(), "TRUNCATE refresh_tokens, webhook_events RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to cleanup tables: %v", err)
	}
	_, err = testDB.Exec(context.Background(), "UPDATE providers SET destination_url = $1, signing_secret = $2 WHERE name = $3", "", "", providers.Stripe)
	if err != nil {
		t.Fatalf("failed to reset tables :%v", err)
	}
}

// getValidLoginCredentials returns an array of bytes containing the json equivalent of a valid login request body
func getValidLoginCredentials() []byte {
	loginCreds := loginRequestBody{Email: config.Envs.UserEmail, Password: config.Envs.UserPassword}
	body, _ := json.Marshal(&loginCreds)
	return body
}

// createAccessToken generates a valid access token and returns it to the user
func createAccessToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Duration(config.Envs.AccessTokenTTL) * time.Minute)
	token, err := testAuthService.GenerateAccessToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// createAccessToken generates a expired access token and returns it to the user
func createExpiredAccessToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Minute * -1)
	token, err := testAuthService.GenerateAccessToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// CreateRefreshToken generates a valid refresh token, saves it to the database and returns it to the user
func createRefreshToken(t *testing.T) string {
	email := config.Envs.UserEmail
	now := time.Now()
	exp := now.Add(time.Duration(config.Envs.RefreshTokenTTL) * time.Hour)
	token, err := testAuthService.GenerateRefreshToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tokenStruct, err := testAuthRepo.Create(ctx, token, exp, now)
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
	token, err := testAuthService.GenerateRefreshToken(email, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tokenStruct, err := testAuthRepo.Create(ctx, token, exp, now)
	if err != nil {
		t.Fatal(err)
	}
	return tokenStruct.Token
}

// getProviderID retrieves the uuid of the passed in provider name
func getProviderID(ctx context.Context, t *testing.T, name string) string {
	query := `SELECT id FROM providers WHERE name = $1`
	var id string
	err := testDB.QueryRow(ctx, query, name).Scan(&id)
	if err != nil {
		t.Errorf("error getting provider ID of %s: %v", name, err)
	}
	return id
}

// getStripeBytesExceeded returns a stripe payload that exceeds the maxBodyBytes defined in the stripe handler
func getStripeBytesExceeded() []byte {
	return bytes.Repeat([]byte("x"), 65540) // exceeds maxBodyBytes (65539)
}

// getStripeInvalidSignaturePayload returns a stripe payload with an invalid signature
func getStripeInvalidSignaturePayload() []byte {
	return []byte(`{
		"id": "evt_1NG8Du2eZvKYlo2CUI79vXWy",
		"object": "event",
		"type": "setup_intent.created"
	}`)
}

// getStripeValidSignatureInvalidBody returns a stripe payload with a valid signature but invalid body
func getStripeValidSignatureInvalidBody() []byte {
	return []byte(`{"id":"evt_bad","object":"event","type":""}`) // passes sig check, fails at insert
}

// getStripeValidPayload returns a valid stripe payload
func getStripeValidPayload() []byte {
	var buf bytes.Buffer
	err := json.Compact(&buf, []byte(`{
		"id": "evt_1NG8Du2eZvKYlo2CUI79vXWy",
		"object": "event",
		"api_version": "2026-04-22.dahlia",
		"created": 1686089970,
		"data": {
			"object": {
				"id": "seti_1NG8Du2eZvKYlo2C9XMqbR0x",
				"object": "setup_intent",
				"status": "requires_confirmation",
				"payment_method_types": ["acss_debit"],
				"livemode": false,
				"metadata": {}
			}
		},
		"livemode": false,
		"pending_webhooks": 0,
		"request": {"id": null, "idempotency_key": null},
		"type": "setup_intent.created"
	}`))
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// insertStripeConfig inserts the default stripe configuration into the test database
func insertStripeConfig(ctx context.Context, t *testing.T, forwardedTo string, secret string) {
	encrypted, err := crypto.EncryptSigningKey(secret)
	if err != nil {
		t.Fatal(err)
	}
	query := `UPDATE providers SET destination_url = $1, signing_secret = $2 WHERE name = $3`
	if _, err := testDB.Exec(ctx, query, forwardedTo, encrypted, providers.Stripe); err != nil {
		t.Fatal(err)
	}
}

// computeStripeSignature returns a stripe signature string using the provided payload and signature raw secret
func computeStripeSignature(payload []byte, signature string) string {
	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(signature)) // use full secret as-is
	if _, err := fmt.Fprintf(mac, "%d", timestamp); err != nil {
		panic(err)
	}
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
