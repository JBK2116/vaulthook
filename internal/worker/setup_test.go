package worker

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

	"github.com/JBK2116/vaulthook/internal/api/handler"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/db"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/logger"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/JBK2116/vaulthook/internal/providers/stripe"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

const stripeSecret = "whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f"
const forwardedTo = "http://localhost:8080/api/webhooks/stripe"

var testDB *pgxpool.Pool
var testLogger *zerolog.Logger

// EVENT HANDLING
var eventRepo *events.EventRepo
var eventService *events.EventService

// PROVIDERS
var testProviderRepo *providers.ProviderRepo
var testProviderService *providers.ProviderService
var testProviderHandler *handler.ProviderHandler

// STRIPE
var stripeRepo *stripe.StripeRepo
var stripeService *stripe.StripeService
var stripeHandle *handler.StripeHandler

// WORKER
var QWorkerRepo WorkerRepository

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
	// configure the event variables
	eventR := events.NewEventRepo(db.DB)
	eventRepo = eventR
	eventS := events.NewEventService(l, eventR)
	eventService = eventS
	// configure the provider variables
	providerR := providers.NewProviderRepo(db.DB)
	testProviderRepo = providerR
	providerS := providers.NewProviderService(providerR)
	testProviderService = providerS
	providerH := handler.NewProviderHandler(l, providerS)
	testProviderHandler = providerH
	// configure the stripe variables
	stripeR := stripe.NewStripeRepo(db.DB)
	stripeRepo = stripeR
	stripeS := stripe.NewStripeService(l, stripeR, providerR)
	stripeService = stripeS
	stripeH := handler.NewStripeHandler(l, stripeS, eventS)
	stripeHandle = stripeH
	// configure the worker variables
	QueueWorkerRepo := NewQueueWorkerRepo(db.DB)
	QWorkerRepo = QueueWorkerRepo
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
func insertStripeConfig(t *testing.T) {
	encrypted, err := crypto.EncryptSigningKey(stripeSecret)
	if err != nil {
		t.Fatal(err)
	}
	query := `UPDATE providers SET destination_url = $1, signing_secret = $2 WHERE name = $3`
	if _, err := testDB.Exec(context.Background(), query, forwardedTo, encrypted, providers.Stripe); err != nil {
		t.Fatal(err)
	}
}

// computeStripeSignature returns a stripe signature string using the provided payload and signature raw secret
func computeStripeSignature() string {
	payload := getStripeValidPayload()
	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(stripeSecret)) // use full secret as-is
	if _, err := fmt.Fprintf(mac, "%d", timestamp); err != nil {
		panic(err)
	}
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
