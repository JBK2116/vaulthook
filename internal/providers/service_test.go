package providers

import (
	"context"
	"os"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var testDB *pgxpool.Pool

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
	code := m.Run()
	os.Exit(code)
}

func beforeEachProviders(t *testing.T) {
	t.Helper()
	// Reset providers to unconfigured state.
	_, err := testDB.Exec(context.Background(),
		`UPDATE providers SET signing_secret = '', destination_url = '', is_configured = false`)
	if err != nil {
		t.Fatalf("failed to reset providers: %v", err)
	}
}

func TestProviderService_GetAll(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	providers, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected at least one provider, got 0")
	}
	// All providers should be unconfigured after reset.
	for _, p := range providers {
		if p.IsConfigured {
			t.Fatalf("expected provider %q to be unconfigured after reset", p.Name)
		}
	}
}

func TestProviderService_Configure_Success(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	// Get the Stripe provider ID.
	providers, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var stripeID string
	for _, p := range providers {
		if p.Name == "Stripe" {
			stripeID = p.ID.String()
			break
		}
	}
	if stripeID == "" {
		t.Fatal("Stripe provider not found")
	}

	prov, err := svc.Configure(ctx, stripeID, "whsec_test123", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !prov.IsConfigured {
		t.Fatal("expected is_configured to be true after Configure")
	}
	if prov.DestinationURL != "https://example.com/webhook" {
		t.Fatalf("expected destination_url, got %q", prov.DestinationURL)
	}
	// Configure returns the encrypted signing_secret (as stored in DB),
	// not the original plaintext.
	if prov.SigningSecret == "" {
		t.Fatal("expected non-empty signing_secret (encrypted)")
	}
}

func TestProviderService_Configure_InvalidUUID(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	_, err := svc.Configure(ctx, "not-a-uuid", "secret", "https://example.com")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestProviderService_Configure_MissingSecret(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	providers, _ := svc.GetAll(ctx)
	stripeID := providers[0].ID.String()

	_, err := svc.Configure(ctx, stripeID, "", "https://example.com")
	if err == nil {
		t.Fatal("expected error for missing signing secret")
	}
	if err != ErrMissingSigningSecret {
		t.Fatalf("expected ErrMissingSigningSecret, got %v", err)
	}
}

func TestProviderService_Configure_MissingDestination(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	providers, _ := svc.GetAll(ctx)
	stripeID := providers[0].ID.String()

	_, err := svc.Configure(ctx, stripeID, "whsec_test", "")
	if err == nil {
		t.Fatal("expected error for missing destination URL")
	}
	if err != ErrMissingDestination {
		t.Fatalf("expected ErrMissingDestination, got %v", err)
	}
}

func TestProviderService_Configure_WhitespaceSecret(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)
	svc := NewProviderService(repo)

	providers, _ := svc.GetAll(ctx)
	stripeID := providers[0].ID.String()

	// Secret that is only whitespace should be treated as empty.
	_, err := svc.Configure(ctx, stripeID, "   ", "https://example.com")
	if err == nil {
		t.Fatal("expected error for whitespace-only signing secret")
	}
}

func TestProviderRepo_GetProviderRouting(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)

	routing, err := repo.GetProviderRouting(ctx, "Stripe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.ID.String() == "" {
		t.Fatal("expected non-empty provider ID")
	}
	// After reset, destination_url should be empty.
	if routing.ForwardedTo != "" {
		t.Fatalf("expected empty destination_url after reset, got %q", routing.ForwardedTo)
	}
}

func TestProviderRepo_GetProviderRouting_Unknown(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)

	_, err := repo.GetProviderRouting(ctx, "UnknownProvider")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestProviderRepo_GetSigningKey(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)

	key, err := repo.GetSigningKey(ctx, "Stripe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After reset, signing_secret should be empty.
	if key != "" {
		t.Fatalf("expected empty signing key after reset, got %q", key)
	}
}

func TestProviderRepo_GetAll(t *testing.T) {
	beforeEachProviders(t)
	ctx := context.Background()

	repo := NewProviderRepo(testDB)

	providers, err := repo.getAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}
	for _, p := range providers {
		if p.Name == "" {
			t.Fatal("expected non-empty provider name")
		}
	}
}
