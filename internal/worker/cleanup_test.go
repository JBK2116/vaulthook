package worker

import (
	"context"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/events"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/rs/zerolog"
)

func TestWorkerPool_Notify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))
	pool := NewWorkerPool(ctx, eventSvc, &l, testDB)

	// Notify should not block even when no workers are actively listening
	// (workers start in background goroutines; the signal channel is buffered).
	pool.Notify()
	pool.Notify()
	pool.Notify()
	// If we reach here without deadlocking, Notify is non-blocking.
}

func TestWorker_Send(t *testing.T) {
	beforeEachWorker(t)

	l := zerolog.Nop()
	eventSvc := events.NewEventService(&l, events.NewEventRepo(testDB))

	// Start the event service so it flushes periodically.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go eventSvc.Start(ctx)

	// Subscribe to verify the event arrives.
	ch, unsub := eventSvc.Subscribe()
	defer unsub()

	repo := NewWorkerRepo(testDB, WorkerKindQueue)
	w := newWorker(eventSvc, repo, &l)

	hook := model.Webhook{
		Provider:       "Stripe",
		EventType:      "test.event",
		DeliveryStatus: model.DeliveryStatusDelivered,
	}
	w.send(&hook)

	// Wait for the event service to flush (ticker fires every 100ms).
	select {
	case batch := <-ch:
		if len(batch) != 1 {
			t.Fatalf("expected 1 event in batch, got %d", len(batch))
		}
		if batch[0].Provider != "Stripe" {
			t.Fatalf("expected Provider 'Stripe', got %q", batch[0].Provider)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sent event")
	}
}

func TestCleanupWorker_RunCleanup_AgeBased(t *testing.T) {
	beforeEachWorker(t)

	l := zerolog.Nop()
	w := NewCleanupWorker(&l, testDB)

	ctx := context.Background()
	provID := getAnyProviderID(ctx, t)

	// Insert an old webhook (received 8 days ago).
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	_, err := testDB.Exec(ctx, `
		INSERT INTO webhook_events
			(provider_id, provider, event_id, event_type, headers, payload, delivery_status, forwarded_to, received_at)
		VALUES ($1, 'Stripe', 'evt_old', 'test.old', '{}', '{}', 'delivered', 'https://example.com', $2)`,
		provID, oldTime)
	if err != nil {
		t.Fatalf("failed to insert old webhook: %v", err)
	}

	// Insert a recent webhook.
	_, err = testDB.Exec(ctx, `
		INSERT INTO webhook_events
			(provider_id, provider, event_id, event_type, headers, payload, delivery_status, forwarded_to, received_at)
		VALUES ($1, 'Stripe', 'evt_new', 'test.new', '{}', '{}', 'delivered', 'https://example.com', NOW())`,
		provID)
	if err != nil {
		t.Fatalf("failed to insert recent webhook: %v", err)
	}

	// Run cleanup should delete the old event.
	w.runCleanup(ctx)

	// The old webhook should be gone.
	var count int
	err = testDB.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events WHERE event_id = 'evt_old'`).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected old webhook to be deleted, got %d remaining", count)
	}

	// The recent webhook should remain.
	err = testDB.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events WHERE event_id = 'evt_new'`).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected recent webhook to remain, got %d", count)
	}
}

func TestCleanupWorker_RunCleanup_EmptyTable(t *testing.T) {
	beforeEachWorker(t)

	l := zerolog.Nop()
	w := NewCleanupWorker(&l, testDB)

	ctx := context.Background()
	// Run cleanup on an empty table. 
	w.runCleanup(ctx)

	// Table should still be empty.
	var count int
	err := testDB.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_events`).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 webhooks after cleanup on empty table, got %d", count)
	}
}

func TestNewCleanupWorker(t *testing.T) {
	l := zerolog.Nop()
	w := NewCleanupWorker(&l, testDB)
	if w == nil {
		t.Fatal("expected non-nil cleanupWorker")
	}
	if w.db == nil {
		t.Fatal("expected non-nil db in cleanupWorker")
	}
	if w.logger == nil {
		t.Fatal("expected non-nil logger in cleanupWorker")
	}
}
