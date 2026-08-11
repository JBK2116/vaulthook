package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func TestForwardEvent_Success(t *testing.T) {
	beforeEachWorker(t)

	// Mock destination that returns 200.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	setProviderDestinationURL(ctx, t, provID, srv.URL)

	hook := insertWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	updates := w.forwardEvent(ctx, []model.Webhook{hook})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	u := updates[0]
	if u.deliveryStatus != model.DeliveryStatusDelivered {
		t.Fatalf("expected 'delivered', got %q", u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 200 {
		t.Fatalf("expected responseCode 200, got %v", u.responseCode)
	}
	if u.nextRetryAt != nil {
		t.Fatal("expected nil nextRetryAt for success")
	}
}

func TestForwardEvent_ClientError(t *testing.T) {
	beforeEachWorker(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	setProviderDestinationURL(ctx, t, provID, srv.URL)

	hook := insertWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	updates := w.forwardEvent(ctx, []model.Webhook{hook})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	u := updates[0]
	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected 'failed', got %q", u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 400 {
		t.Fatalf("expected responseCode 400, got %v", u.responseCode)
	}
	if u.nextRetryAt != nil {
		t.Fatal("expected nil nextRetryAt for non-retryable 4xx")
	}
}

func TestForwardEvent_ServerError(t *testing.T) {
	beforeEachWorker(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	setProviderDestinationURL(ctx, t, provID, srv.URL)

	hook := insertWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	updates := w.forwardEvent(ctx, []model.Webhook{hook})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	u := updates[0]
	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected 'failed', got %q", u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 500 {
		t.Fatalf("expected responseCode 500, got %v", u.responseCode)
	}
	if u.nextRetryAt == nil {
		t.Fatal("expected non-nil nextRetryAt for retryable 5xx")
	}
}

func TestForwardEvent_RateLimited(t *testing.T) {
	beforeEachWorker(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	setProviderDestinationURL(ctx, t, provID, srv.URL)

	hook := insertWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	updates := w.forwardEvent(ctx, []model.Webhook{hook})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	u := updates[0]
	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected 'failed', got %q", u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 429 {
		t.Fatalf("expected responseCode 429, got %v", u.responseCode)
	}
	if u.nextRetryAt == nil {
		t.Fatal("expected non-nil nextRetryAt for rate-limited response")
	}
}

func TestForwardEvent_ConnectionRefused(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	// Point to a closed server to simulate connection refused.
	setProviderDestinationURL(ctx, t, provID, "http://127.0.0.1:1")

	hook := insertWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	updates := w.forwardEvent(ctx, []model.Webhook{hook})
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	u := updates[0]
	if u.lastError == nil {
		t.Fatal("expected transport error in lastError")
	}
	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected 'failed', got %q", u.deliveryStatus)
	}
	if u.nextRetryAt == nil {
		t.Fatal("expected non-nil nextRetryAt for transport failure")
	}
}

func TestForwardEvent_StripeHeadersForwarded(t *testing.T) {
	beforeEachWorker(t)

	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	setProviderDestinationURL(ctx, t, provID, srv.URL)

	// Insert a Stripe webhook with headers.
	hook := insertStripeWebhook(ctx, t, provID, "processing")

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	_ = w.forwardEvent(ctx, []model.Webhook{hook})
	if receivedHeaders.Get("Stripe-Signature") != "t=123,v1=abc" {
		t.Fatalf("expected Stripe-Signature to be forwarded, got %q", receivedHeaders.Get("Stripe-Signature"))
	}
	if receivedHeaders.Get("Authorization") != "" {
		t.Fatal("Authorization should NOT be forwarded")
	}
}

// helpers

func setProviderDestinationURL(ctx context.Context, t *testing.T, provID uuid.UUID, url string) {
	t.Helper()
	var prov model.Provider
	err := testDB.QueryRow(ctx, `
		UPDATE providers SET destination_url = $1 WHERE id = $2
		RETURNING id, name, signing_secret, destination_url,
		          max_retries, max_req_second, is_configured, created_at`,
		url, provID).Scan(
		&prov.ID, &prov.Name, &prov.SigningSecret,
		&prov.DestinationURL, &prov.MaxRetries, &prov.MaxReqSecond,
		&prov.IsConfigured, &prov.CreatedAt,
	)
	if err != nil {
		t.Fatalf("failed to set destination URL: %v", err)
	}
	// Update the Redis-backed provider cache so forwardEvent picks up the new URL.
	if err := providers.Cache.Set(ctx, model.ProviderName(prov.Name), prov); err != nil {
		t.Fatalf("failed to update provider cache: %v", err)
	}
}

func insertStripeWebhook(ctx context.Context, t *testing.T, provID uuid.UUID, status string) model.Webhook {
	t.Helper()
	var w model.Webhook
	headers := `{"Content-Type":["application/json"],"Stripe-Signature":["t=123,v1=abc"],"Authorization":["Bearer secret"]}`
	query := `
		INSERT INTO webhook_events
			(provider_id, provider, event_id, event_type, headers, payload, delivery_status, forwarded_to, received_at)
		VALUES ($1, 'Stripe', 'evt_test', 'test.event', $2, '{}', $3, 'https://example.com', NOW())
		RETURNING id, provider_id, provider, event_id, event_type, headers, payload,
		          delivery_status, forwarded_to, response_code, retry_count,
		          next_retry_at, last_error, received_at, created_at, updated_at
	`
	err := testDB.QueryRow(ctx, query, provID, headers, status).Scan(
		&w.ID, &w.ProviderID, &w.Provider, &w.EventID, &w.EventType,
		&w.Headers, &w.Payload, &w.DeliveryStatus, &w.ForwardedTo,
		&w.ResponseCode, &w.RetryCount, &w.NextRetryAt, &w.LastError,
		&w.ReceivedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert test webhook: %v", err)
	}
	return w
}
