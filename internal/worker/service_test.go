package worker

import (
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/model"
)

func init() {
	config.Envs.RetryIntervalSeconds = 15
}

func TestSetDefaultUpdateValues(t *testing.T) {
	var u updateWebhook
	errMsg := "connection refused"
	setDefaultUpdateValues(errMsg, &u)

	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected status %q, got %q", model.DeliveryStatusFailed, u.deliveryStatus)
	}
	if u.lastError == nil || *u.lastError != errMsg {
		t.Fatalf("expected lastError %q, got %v", errMsg, u.lastError)
	}
	if u.nextRetryAt == nil {
		t.Fatal("expected non-nil nextRetryAt")
	}
	if u.responseCode != nil {
		t.Fatalf("expected nil responseCode, got %v", *u.responseCode)
	}
}

func TestSetSuccessUpdateValues(t *testing.T) {
	var u updateWebhook
	setSuccessUpdateValues(200, &u)

	if u.deliveryStatus != model.DeliveryStatusDelivered {
		t.Fatalf("expected status %q, got %q", model.DeliveryStatusDelivered, u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 200 {
		t.Fatalf("expected responseCode 200, got %v", u.responseCode)
	}
	if u.lastError != nil {
		t.Fatalf("expected nil lastError, got %v", *u.lastError)
	}
	if u.nextRetryAt != nil {
		t.Fatal("expected nil nextRetryAt for success")
	}
}

func TestSetFailureUpdateValues(t *testing.T) {
	var u updateWebhook
	errMsg := "400 Bad Request"
	setFailureUpdateValues(400, errMsg, &u)

	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected status %q, got %q", model.DeliveryStatusFailed, u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 400 {
		t.Fatalf("expected responseCode 400, got %v", u.responseCode)
	}
	if u.lastError == nil || *u.lastError != errMsg {
		t.Fatalf("expected lastError %q, got %v", errMsg, u.lastError)
	}
	if u.nextRetryAt != nil {
		t.Fatal("expected nil nextRetryAt for non-retryable failure")
	}
}

func TestSetRetryableUpdateValues(t *testing.T) {
	var u updateWebhook
	errMsg := "500 Internal Server Error"
	setRetryableUpdateValues(500, errMsg, &u)

	if u.deliveryStatus != model.DeliveryStatusFailed {
		t.Fatalf("expected status %q, got %q", model.DeliveryStatusFailed, u.deliveryStatus)
	}
	if u.responseCode == nil || *u.responseCode != 500 {
		t.Fatalf("expected responseCode 500, got %v", u.responseCode)
	}
	if u.lastError == nil || *u.lastError != errMsg {
		t.Fatalf("expected lastError %q, got %v", errMsg, u.lastError)
	}
	if u.nextRetryAt == nil {
		t.Fatal("expected non-nil nextRetryAt for retryable failure")
	}
}

func TestSetRateLimitedUpdateValues(t *testing.T) {
	t.Run("with Retry-After header", func(t *testing.T) {
		var u updateWebhook
		setRateLimitedUpdateValues(429, "rate limited", "30", &u)

		if u.deliveryStatus != model.DeliveryStatusFailed {
			t.Fatalf("expected status %q, got %q", model.DeliveryStatusFailed, u.deliveryStatus)
		}
		if u.nextRetryAt == nil {
			t.Fatal("expected non-nil nextRetryAt")
		}
		// Should be roughly 30 seconds from now.
		expected := time.Now().Add(30 * time.Second)
		diff := u.nextRetryAt.Sub(expected)
		if diff < -time.Second || diff > time.Second {
			t.Fatalf("expected nextRetryAt ~30s from now, got %v (diff: %v)", u.nextRetryAt, diff)
		}
	})

	t.Run("without Retry-After header", func(t *testing.T) {
		var u updateWebhook
		setRateLimitedUpdateValues(429, "rate limited", "", &u)

		if u.nextRetryAt == nil {
			t.Fatal("expected non-nil nextRetryAt")
		}
		// Falls back to config.Envs.RetryIntervalSeconds (15).
		expected := time.Now().Add(15 * time.Second)
		diff := u.nextRetryAt.Sub(expected)
		if diff < -time.Second || diff > time.Second {
			t.Fatalf("expected nextRetryAt ~15s from now, got %v (diff: %v)", u.nextRetryAt, diff)
		}
	})

	t.Run("with invalid Retry-After header", func(t *testing.T) {
		var u updateWebhook
		setRateLimitedUpdateValues(503, "unavailable", "not-a-number", &u)

		if u.nextRetryAt == nil {
			t.Fatal("expected non-nil nextRetryAt")
		}
		// Falls back to config.Envs.RetryIntervalSeconds (15).
		expected := time.Now().Add(15 * time.Second)
		diff := u.nextRetryAt.Sub(expected)
		if diff < -time.Second || diff > time.Second {
			t.Fatalf("expected nextRetryAt ~15s from now, got %v (diff: %v)", u.nextRetryAt, diff)
		}
	})
}
