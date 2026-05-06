package worker

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/providers"
	"github.com/jackc/pgx/v5"
)

func TestQueueWorkerRepository(t *testing.T) {
	validPayload := getStripeValidPayload()
	secret := computeStripeSignature()
	tests := map[string]struct {
		err            error
		shouldAddHooks bool
	}{
		"no webhooks found": {err: pgx.ErrNoRows, shouldAddHooks: false},
		"webhooks found":    {err: nil, shouldAddHooks: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			insertStripeConfig(t)
			var hooks []*providers.Webhook
			if test.shouldAddHooks {
				// insert two webhooks into the database.
				for range 2 {
					url := "http://localhost:8080/api/webhooks/stripe"
					r := httptest.NewRequest("POST", url, bytes.NewBuffer(validPayload))
					r.Header.Set("Content-Type", "application/json")
					r.Header.Set("Stripe-Signature", secret)
					w := httptest.NewRecorder()
					stripeHandle.Receive(w, r)
					if w.Result().StatusCode != http.StatusOK {
						t.Fatalf("handler call failed: %d", w.Result().StatusCode)
					}
				}
			}
			// retrieve the first webhook oldest by received_at.
			hook, err := QWorkerRepo.GetEvent(ctx)
			if test.shouldAddHooks {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, test.err) {
					t.Fatalf("expected error: %v, got: %v", test.err, err)
				}
			}
			if test.shouldAddHooks {
				hooks = append(hooks, hook)
				// retrieve the second webhook.
				h, err := QWorkerRepo.GetEvent(ctx)
				if err != nil {
					t.Fatalf("unexpected database error: %v", err)
				}
				hooks = append(hooks, h)
				// webhooks must not be the same.
				if hooks[0].ID == hooks[1].ID {
					t.Fatalf("expected different webhook IDs, got same ID: %s", hooks[0].ID)
				}
				// queue worker orders by received_at ASC. If it is equal then ID is the tie breaker.
				if hooks[0].ReceivedAt.After(hooks[1].ReceivedAt) {
					t.Fatalf("expected first webhook to have older or equal received_at than second")
				}
				if hooks[0].ReceivedAt.Equal(hooks[1].ReceivedAt) && hooks[0].ID.String() > hooks[1].ID.String() {
					t.Fatalf("expected first webhook ID to be less than second when received_at is equal")
				}
			}
			afterEach(t)
		})
	}
}

func TestRetryWorkerRepo(t *testing.T) {
	validPayload := getStripeValidPayload()
	secret := computeStripeSignature()
	tests := map[string]struct {
		err            error
		shouldAddHooks bool
	}{
		"no webhooks found": {err: pgx.ErrNoRows, shouldAddHooks: false},
		"webhooks found":    {err: nil, shouldAddHooks: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			insertStripeConfig(t)
			var hooks []*providers.Webhook
			if test.shouldAddHooks {
				// insert two webhooks into the database.
				for range 2 {
					url := "http://localhost:8080/api/webhooks/stripe"
					r := httptest.NewRequest("POST", url, bytes.NewBuffer(validPayload))
					r.Header.Set("Content-Type", "application/json")
					r.Header.Set("Stripe-Signature", secret)
					w := httptest.NewRecorder()
					stripeHandle.Receive(w, r)
					if w.Result().StatusCode != http.StatusOK {
						t.Fatalf("handler call failed: %d", w.Result().StatusCode)
					}
				}
				// mark all inserted webhooks as retrying.
				setAllAsRetry(t)
			}
			// retrieve the first webhook most overdue by next_retry_at.
			hook, err := RWorkerRepo.GetEvent(ctx)
			if test.shouldAddHooks {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if !errors.Is(err, test.err) {
					t.Fatalf("expected error: %v, got: %v", test.err, err)
				}
			}
			if test.shouldAddHooks {
				hooks = append(hooks, hook)
				// retrieve the second webhook.
				h, err := RWorkerRepo.GetEvent(ctx)
				if err != nil {
					t.Fatalf("unexpected database error: %v", err)
				}
				hooks = append(hooks, h)
				// webhooks must not be the same.
				if hooks[0].ID == hooks[1].ID {
					t.Fatalf("expected different webhook IDs, got same ID: %s", hooks[0].ID)
				}
				// retry worker orders by next_retry_at ASC first must not be after second.
				if hooks[0].NextRetryAt != nil && hooks[1].NextRetryAt != nil {
					if hooks[0].NextRetryAt.After(*hooks[1].NextRetryAt) {
						t.Fatalf("expected first webhook next_retry_at to be before or equal to second")
					}
				}
			}
			afterEach(t)
		})
	}
}
