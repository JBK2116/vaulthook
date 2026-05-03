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
				// Insert two webhooks into the database.
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
			// retrieve the "first" webhook.
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
			// test uniqueness and fifo.
			if test.shouldAddHooks {
				// append the first webhook
				hooks = append(hooks, hook)
				// receive and append the "second" webhook
				h, err := QWorkerRepo.GetEvent(ctx)
				if err != nil {
					t.Fatalf("unexpected database error: %v", err)
				}
				hooks = append(hooks, h)
				// webhooks must not be the same
				if hooks[0].ID == hooks[1].ID {
					t.Fatalf("expected webhook IDs to be different. Received different IDs")
				}
				// first queried webhook must be older than second
				if !hooks[0].CreatedAt.Before(hooks[1].CreatedAt) {
					t.Fatalf("expected first webhook to be older than second webhook")
				}
			}
		})
	}
}
