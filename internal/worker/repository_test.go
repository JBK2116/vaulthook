package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestGetEvent_QueueWorker_PicksUpQueuedEvent(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	// Insert a queued webhook.
	provID := getAnyProviderID(ctx, t)
	insertWebhook(ctx, t, provID, "queued")

	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	hook, err := repo.GetEvent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.DeliveryStatus != model.DeliveryStatusProcessing {
		t.Fatalf("expected status 'processing', got %q", hook.DeliveryStatus)
	}
}

func TestGetEvent_QueueWorker_NoEvents(t *testing.T) {
	beforeEachWorker(t)

	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	_, err := repo.GetEvent(context.Background())
	if err == nil {
		t.Fatal("expected error when no queued events exist")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestGetEvent_RetryWorker_PicksUpFailedEvent(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	insertWebhookWithStatus(ctx, t, provID, "failed", time.Now().Add(-time.Minute), 0)

	repo := NewWorkerRepo(testDB, WorkerKindRetry)
	hook, err := repo.GetEvent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.DeliveryStatus != model.DeliveryStatusRetrying {
		t.Fatalf("expected status 'retrying', got %q", hook.DeliveryStatus)
	}
}

func TestGetEvent_RetryWorker_ExhaustedRetries(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	// Insert event with retry_count exceeding provider max_retries (5).
	// The subquery on providers.max_retries filters it out.
	insertWebhookWithStatus(ctx, t, provID, "failed", time.Now().Add(-time.Minute), 5)

	repo := NewWorkerRepo(testDB, WorkerKindRetry)
	_, err := repo.GetEvent(ctx)
	if err == nil {
		t.Fatal("expected error when retries are exhausted")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for exhausted retries, got %v", err)
	}
}

func TestGetEvent_ReplayWorker_PicksUpReplayingEvent(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	insertWebhookWithStatus(ctx, t, provID, "replaying", time.Now(), 0)

	repo := NewWorkerRepo(testDB, WorkerKindReplay)
	hook, err := repo.GetEvent(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.DeliveryStatus != model.DeliveryStatusRetrying { // Replay sets to 'replaying' which is the same
		// Actually replay worker SETs 'replaying' and SELECTs 'replaying'. Verify it was returned.
		if hook.DeliveryStatus != "replaying" {
			t.Fatalf("expected 'replaying', got %q", hook.DeliveryStatus)
		}
	}
	_ = hook
}

func TestUpdateEvent(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	hook := insertWebhook(ctx, t, provID, "processing")

	code := 200
	forwardedTo := "https://example.com"
	updates := updateWebhook{
		id:             hook.ID,
		deliveryStatus: model.DeliveryStatusDelivered,
		responseCode:   &code,
		lastError:      nil,
		nextRetryAt:    nil,
		forwardedTo:    &forwardedTo,
	}

	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	updated, err := repo.UpdateEvent(ctx, updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.DeliveryStatus != model.DeliveryStatusDelivered {
		t.Fatalf("expected 'delivered', got %q", updated.DeliveryStatus)
	}
	if updated.ResponseCode == nil || *updated.ResponseCode != 200 {
		t.Fatalf("expected responseCode 200, got %v", updated.ResponseCode)
	}
}

func TestUpdateEvent_RetryWorker_IncrementsRetryCount(t *testing.T) {
	beforeEachWorker(t)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)
	hook := insertWebhook(ctx, t, provID, "retrying")

	nextRetry := time.Now().Add(time.Minute)
	errMsg := "temporary failure"
	forwardedTo := "https://example.com"
	updates := updateWebhook{
		id:             hook.ID,
		deliveryStatus: model.DeliveryStatusFailed,
		lastError:      &errMsg,
		nextRetryAt:    &nextRetry,
		forwardedTo:    &forwardedTo,
	}

	repo := NewWorkerRepo(testDB, WorkerKindRetry)
	updated, err := repo.UpdateEvent(ctx, updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.RetryCount != 1 {
		t.Fatalf("expected retryCount 1, got %d", updated.RetryCount)
	}
	if updated.LastError == nil || *updated.LastError != errMsg {
		t.Fatalf("expected lastError, got %v", updated.LastError)
	}
}

// helpers

func insertWebhook(ctx context.Context, t *testing.T, provID uuid.UUID, status string) model.Webhook {
	t.Helper()
	return insertWebhookWithStatus(ctx, t, provID, status, time.Now(), 0)
}

func insertWebhookWithStatus(ctx context.Context, t *testing.T, provID uuid.UUID, status string, nextRetryAt time.Time, retryCount int) model.Webhook {
	t.Helper()
	var w model.Webhook
	var nextRetry any
	if nextRetryAt.IsZero() {
		nextRetry = nil
	} else {
		nextRetry = nextRetryAt
	}
	query := `
		INSERT INTO webhook_events
			(provider_id, provider, event_id, event_type, headers, payload, delivery_status, forwarded_to, next_retry_at, retry_count, received_at)
		VALUES ($1, 'Stripe', 'evt_test', 'test.event', '{}', '{}', $2, 'https://example.com', $3, $4, NOW())
		RETURNING id, provider_id, provider, event_id, event_type, headers, payload,
		          delivery_status, forwarded_to, response_code, retry_count,
		          next_retry_at, last_error, received_at, created_at, updated_at
	`
	err := testDB.QueryRow(ctx, query, provID, status, nextRetry, retryCount).Scan(
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

func getAnyProviderID(ctx context.Context, t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx, `SELECT id FROM providers WHERE name = 'Stripe'`).Scan(&id)
	if err != nil {
		t.Fatalf("no Stripe provider found: %v", err)
	}
	return id
}
