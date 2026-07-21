package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../../.env"); err != nil {
		panic(err)
	}
	config.Init()
	ctx := context.Background()
	db, err := config.NewPG(ctx)
	if err != nil {
		panic(err)
	}
	testDB = db.DB
	code := m.Run()
	os.Exit(code)
}

func beforeEachGithub(t *testing.T) {
	t.Helper()
	// Reset GitHub provider to unconfigured state.
	_, err := testDB.Exec(context.Background(),
		`UPDATE providers SET signing_secret = '', destination_url = '', is_configured = false WHERE name = $1`,
		string(model.Github))
	if err != nil {
		t.Fatalf("failed to reset github provider: %v", err)
	}
}

func setGithubSigningSecret(ctx context.Context, t *testing.T, secret string) {
	t.Helper()
	encrypted, err := crypto.EncryptSigningKey(secret)
	if err != nil {
		t.Fatalf("failed to encrypt secret: %v", err)
	}
	_, err = testDB.Exec(ctx,
		`UPDATE providers SET signing_secret = $1, is_configured = true WHERE name = $2`,
		encrypted, string(model.Github))
	if err != nil {
		t.Fatalf("failed to set github secret: %v", err)
	}
}

func computeGitHubSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidateSecret_Valid(t *testing.T) {
	beforeEachGithub(t)

	ctx := context.Background()
	secret := "github_test_secret_12345"
	setGithubSigningSecret(ctx, t, secret)

	l := zerolog.Nop()
	eventRepo := events.NewEventRepo(testDB)
	providerRepo := providers.NewProviderRepo(testDB)
	svc := NewGitService(&l, eventRepo, providerRepo)

	payload := []byte(`{"action":"opened"}`)
	signature := computeGitHubSignature(secret, payload)

	err := svc.ValidateSecret(ctx, signature, payload)
	if err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestValidateSecret_InvalidSignature(t *testing.T) {
	beforeEachGithub(t)

	ctx := context.Background()
	secret := "github_test_secret_12345"
	setGithubSigningSecret(ctx, t, secret)

	l := zerolog.Nop()
	eventRepo := events.NewEventRepo(testDB)
	providerRepo := providers.NewProviderRepo(testDB)
	svc := NewGitService(&l, eventRepo, providerRepo)

	payload := []byte(`{"action":"opened"}`)
	wrongSig := computeGitHubSignature("wrong_secret", payload)

	err := svc.ValidateSecret(ctx, wrongSig, payload)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestValidateSecret_EmptyPayload(t *testing.T) {
	beforeEachGithub(t)

	ctx := context.Background()
	secret := "github_test_secret_12345"
	setGithubSigningSecret(ctx, t, secret)

	l := zerolog.Nop()
	eventRepo := events.NewEventRepo(testDB)
	providerRepo := providers.NewProviderRepo(testDB)
	svc := NewGitService(&l, eventRepo, providerRepo)

	signature := computeGitHubSignature(secret, []byte{})

	err := svc.ValidateSecret(ctx, signature, []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertWebhook_Success(t *testing.T) {
	beforeEachGithub(t)

	ctx := context.Background()
	secret := "github_test_secret_12345"
	setGithubSigningSecret(ctx, t, secret)

	// Set a destination URL for GitHub so InsertWebhook can resolve routing.
	_, err := testDB.Exec(ctx,
		`UPDATE providers SET destination_url = $1 WHERE name = $2`,
		"https://example.com/webhooks/github", string(model.Github))
	if err != nil {
		t.Fatalf("failed to set destination URL: %v", err)
	}

	l := zerolog.Nop()
	eventRepo := events.NewEventRepo(testDB)
	providerRepo := providers.NewProviderRepo(testDB)
	svc := NewGitService(&l, eventRepo, providerRepo)

	headers := []byte(`{"X-GitHub-Event":["push"],"Content-Type":["application/json"]}`)
	payload := []byte(`{"action":"opened","ref":"refs/heads/main"}`)

	hook, err := svc.InsertWebhook(ctx, headers, payload, "12345", "push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.Provider != string(model.Github) {
		t.Fatalf("expected provider Github, got %q", hook.Provider)
	}
	if hook.EventType != "push" {
		t.Fatalf("expected event_type 'push', got %q", hook.EventType)
	}
	if hook.EventID == nil || *hook.EventID != "12345" {
		t.Fatalf("expected event_id '12345', got %v", hook.EventID)
	}
	if hook.DeliveryStatus != model.DeliveryStatusQueued {
		t.Fatalf("expected status 'queued', got %q", hook.DeliveryStatus)
	}
	if hook.ForwardedTo != "https://example.com/webhooks/github" {
		t.Fatalf("expected forwarded_to, got %q", hook.ForwardedTo)
	}
}

func TestInsertWebhook_NoDestinationURL(t *testing.T) {
	beforeEachGithub(t)

	ctx := context.Background()
	// Don't set destination URL — routing should still work (returns empty URL).
	// But InsertWebhook uses the returned routing.ForwardedTo as-is.

	l := zerolog.Nop()
	eventRepo := events.NewEventRepo(testDB)
	providerRepo := providers.NewProviderRepo(testDB)
	svc := NewGitService(&l, eventRepo, providerRepo)

	headers := []byte(`{}`)
	payload := []byte(`{}`)

	// Should succeed — empty destination URL is allowed at insert time.
	hook, err := svc.InsertWebhook(ctx, headers, payload, "evt1", "issues")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.ForwardedTo != "" {
		t.Fatalf("expected empty forwarded_to, got %q", hook.ForwardedTo)
	}
}
