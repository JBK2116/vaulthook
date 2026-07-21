package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/google/uuid"
)

func TestEventService_SendAndSubscribe(t *testing.T) {
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	ch, unsub := svc.Subscribe()
	defer unsub()

	hook := model.Webhook{
		ID:             uuid.New(),
		Provider:       "Stripe",
		EventType:      "test.event",
		DeliveryStatus: model.DeliveryStatusQueued,
	}
	svc.Send(hook)
	// Manually flush since Start() isn't running.
	svc.drainBroadcast()
	svc.flush()

	select {
	case batch := <-ch:
		if len(batch) != 1 {
			t.Fatalf("expected 1 event in batch, got %d", len(batch))
		}
		if batch[0].ID != hook.ID {
			t.Fatalf("expected hook ID %v, got %v", hook.ID, batch[0].ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventService_MultipleSubscribers(t *testing.T) {
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	ch1, unsub1 := svc.Subscribe()
	defer unsub1()
	ch2, unsub2 := svc.Subscribe()
	defer unsub2()

	hook := model.Webhook{ID: uuid.New(), Provider: "Stripe", DeliveryStatus: model.DeliveryStatusQueued}
	svc.Send(hook)
	svc.drainBroadcast()
	svc.flush()

	// Both subscribers should receive the batch.
	for i, ch := range []<-chan []model.Webhook{ch1, ch2} {
		select {
		case batch := <-ch:
			if len(batch) != 1 {
				t.Fatalf("subscriber %d: expected 1 event, got %d", i, len(batch))
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestEventService_Unsubscribe(t *testing.T) {
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	ch, unsub := svc.Subscribe()
	unsub()

	// After unsub, flush should not send to closed channel.
	hook := model.Webhook{ID: uuid.New(), Provider: "Stripe", DeliveryStatus: model.DeliveryStatusQueued}
	svc.Send(hook)
	svc.drainBroadcast()
	svc.flush()

	// The channel is closed; reading should return zero value immediately.
	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel after unsubscribe")
	}
}

func TestEventService_MaxBatchSize(t *testing.T) {
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	ch, unsub := svc.Subscribe()
	defer unsub()

	// Send more than maxBatchSize events.
	for i := range maxBatchSize + 50 {
		svc.Send(model.Webhook{ID: uuid.New(), Provider: "Stripe", DeliveryStatus: model.DeliveryStatusQueued, EventType: string(rune(i))})
	}
	svc.drainBroadcast()
	svc.flush()

	select {
	case batch := <-ch:
		if len(batch) > maxBatchSize {
			t.Fatalf("batch size %d exceeds max %d", len(batch), maxBatchSize)
		}
		// Some events should have been dropped.
		if dropped := svc.Dropped(); dropped <= 0 {
			t.Fatalf("expected dropped > 0 when exceeding maxBatchSize, got %d", dropped)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for batch")
	}
}

func TestEventService_DroppedResets(t *testing.T) {
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	ch, unsub := svc.Subscribe()
	defer unsub()

	for range maxBatchSize + 10 {
		svc.Send(model.Webhook{ID: uuid.New(), Provider: "Stripe", DeliveryStatus: model.DeliveryStatusQueued})
	}
	svc.drainBroadcast()
	svc.flush()
	<-ch // consume

	first := svc.Dropped()
	if first <= 0 {
		t.Fatalf("expected dropped > 0, got %d", first)
	}
	second := svc.Dropped()
	if second != 0 {
		t.Fatalf("expected Dropped to reset to 0 after first call, got %d", second)
	}
}

func TestEventService_SendNonBlocking(t *testing.T) {
	// Verify Send doesn't block when the broadcast channel is full.
	repo := NewEventRepo(testDB)
	svc := NewEventService(testLogger, repo)

	// Fill the broadcast channel to capacity (10000).
	hook := model.Webhook{ID: uuid.New(), Provider: "Stripe", DeliveryStatus: model.DeliveryStatusQueued}
	for range 10001 {
		svc.Send(hook) // must not block
	}
	// If we got here without deadlocking, Send is non-blocking.
}

// Repository tests

func TestEventRepo_InsertWebhook(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	provID := getProviderID(ctx, t)
	params := model.CreateWebhookParams{
		ProviderID:  provID,
		Provider:    "Stripe",
		EventType:   "charge.succeeded",
		Headers:     []byte(`{"Content-Type":["application/json"]}`),
		Payload:     []byte(`{"id":"evt_test"}`),
		ForwardedTo: "https://example.com/webhook",
		ReceivedAt:  time.Now().UTC(),
	}
	hook, err := NewEventRepo(testDB).InsertWebhook(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hook.DeliveryStatus != model.DeliveryStatusQueued {
		t.Fatalf("expected status 'queued', got %q", hook.DeliveryStatus)
	}
	if hook.Provider != "Stripe" {
		t.Fatalf("expected provider 'Stripe', got %q", hook.Provider)
	}
	if hook.ID == uuid.Nil {
		t.Fatal("expected non-nil UUID")
	}
}

func TestEventRepo_GetAll(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	// Insert two webhooks.
	params := model.CreateWebhookParams{
		ProviderID: provID, Provider: "Stripe", EventType: "test",
		Headers: []byte(`{}`), Payload: []byte(`{}`),
		ForwardedTo: "https://example.com", ReceivedAt: time.Now().UTC(),
	}
	if _, err := repo.InsertWebhook(ctx, params); err != nil {
		t.Fatalf("insert 1 failed: %v", err)
	}
	if _, err := repo.InsertWebhook(ctx, params); err != nil {
		t.Fatalf("insert 2 failed: %v", err)
	}

	hooks, err := repo.getAll(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}
}

func TestEventRepo_GetAll_WithCursor(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	params := model.CreateWebhookParams{
		ProviderID: provID, Provider: "Stripe", EventType: "test",
		Headers: []byte(`{}`), Payload: []byte(`{}`),
		ForwardedTo: "https://example.com", ReceivedAt: time.Now().UTC(),
	}
	if _, err := repo.InsertWebhook(ctx, params); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Cursor in the past should return no results for events just created.
	past := time.Now().Add(-time.Hour)
	hooks, err := repo.getAll(ctx, &past)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hooks) != 0 {
		t.Fatalf("expected 0 hooks with past cursor, got %d", len(hooks))
	}
}

func TestEventRepo_GetStats(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	// Insert hooks with different statuses.
	insertWithStatus := func(status model.DeliveryStatus) {
		_, err := testDB.Exec(ctx, `
			INSERT INTO webhook_events (provider_id, provider, event_type, headers, payload, delivery_status, forwarded_to, received_at)
			VALUES ($1, 'Stripe', 'test', '{}', '{}', $2, 'https://example.com', NOW())`,
			provID, status)
		if err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}
	insertWithStatus(model.DeliveryStatusDelivered)
	insertWithStatus(model.DeliveryStatusDelivered)
	insertWithStatus(model.DeliveryStatusFailed)
	insertWithStatus(model.DeliveryStatusQueued)

	stats, err := repo.getStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Delivered != 2 {
		t.Fatalf("expected 2 delivered, got %d", stats.Delivered)
	}
	if stats.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", stats.Failed)
	}
	if stats.Queued != 1 {
		t.Fatalf("expected 1 queued, got %d", stats.Queued)
	}
}

func TestEventRepo_ReplayEvent(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	params := model.CreateWebhookParams{
		ProviderID: provID, Provider: "Stripe", EventType: "test",
		Headers: []byte(`{}`), Payload: []byte(`{}`),
		ForwardedTo: "https://example.com", ReceivedAt: time.Now().UTC(),
	}
	hook, err := repo.InsertWebhook(ctx, params)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	err = repo.replayEvent(ctx, hook.ID)
	if err != nil {
		t.Fatalf("replayEvent failed: %v", err)
	}

	// Verify status changed to 'replaying'.
	var status string
	err = testDB.QueryRow(ctx, `SELECT delivery_status FROM webhook_events WHERE id = $1`, hook.ID).Scan(&status)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if status != "replaying" {
		t.Fatalf("expected 'replaying', got %q", status)
	}
}

func TestEventService_GetAll(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	params := model.CreateWebhookParams{
		ProviderID: provID, Provider: "Stripe", EventType: "test",
		Headers: []byte(`{}`), Payload: []byte(`{}`),
		ForwardedTo: "https://example.com", ReceivedAt: time.Now().UTC(),
	}
	if _, err := repo.InsertWebhook(ctx, params); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	svc := NewEventService(testLogger, repo)
	hooks, err := svc.GetAll(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooks))
	}
}

func TestEventService_GetStats(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	// Insert one delivered event.
	_, err := testDB.Exec(ctx, `
		INSERT INTO webhook_events (provider_id, provider, event_type, headers, payload, delivery_status, forwarded_to, received_at)
		VALUES ($1, 'Stripe', 'test', '{}', '{}', 'delivered', 'https://example.com', NOW())`, provID)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	svc := NewEventService(testLogger, repo)
	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Delivered != 1 {
		t.Fatalf("expected 1 delivered, got %d", stats.Delivered)
	}
}

func TestEventService_ReplayEvent_ValidUUID(t *testing.T) {
	beforeEachEvents(t)

	ctx := context.Background()
	repo := NewEventRepo(testDB)
	provID := getProviderID(ctx, t)

	params := model.CreateWebhookParams{
		ProviderID: provID, Provider: "Stripe", EventType: "test",
		Headers: []byte(`{}`), Payload: []byte(`{}`),
		ForwardedTo: "https://example.com", ReceivedAt: time.Now().UTC(),
	}
	hook, err := repo.InsertWebhook(ctx, params)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	svc := NewEventService(testLogger, repo)
	err = svc.ReplayEvent(ctx, hook.ID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventService_ReplayEvent_InvalidUUID(t *testing.T) {
	beforeEachEvents(t)

	svc := NewEventService(testLogger, NewEventRepo(testDB))
	err := svc.ReplayEvent(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	if !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
}
