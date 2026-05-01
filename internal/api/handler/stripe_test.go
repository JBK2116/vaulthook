package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Change this accordingly to your development environment. No need to hide it as prod secret is different.
const stripeSecret = "whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f"
const invalidStripeSecret = "whsec_5b5dd0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f"

func TestStripeReceive(t *testing.T) {
	validPayload := getStripeValidPayload()

	tests := map[string]struct {
		payload    []byte
		statusCode int
		secret     string
	}{
		"max bytes exceeded":   {payload: getStripeBytesExceeded(), statusCode: http.StatusServiceUnavailable, secret: computeStripeSignature(getStripeBytesExceeded(), invalidStripeSecret)},
		"invalid signature":    {payload: getStripeInvalidSignaturePayload(), statusCode: http.StatusBadRequest, secret: computeStripeSignature(getStripeInvalidSignaturePayload(), invalidStripeSecret)},
		"invalid body":         {payload: getStripeValidSignatureInvalidBody(), statusCode: http.StatusBadRequest, secret: computeStripeSignature(getStripeValidSignatureInvalidBody(), stripeSecret)},
		"valid signature body": {payload: validPayload, statusCode: http.StatusOK, secret: computeStripeSignature(validPayload, stripeSecret)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			url := "http://localhost:8080/api/webhooks/stripe"
			insertStripeConfig(ctx, t, url, stripeSecret)
			r := httptest.NewRequest("POST", url, bytes.NewBuffer(test.payload))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Stripe-Signature", test.secret)
			w := httptest.NewRecorder()
			stripeHandle.receive(w, r)
			if test.statusCode != w.Result().StatusCode {
				t.Fatalf("expected %d, received %d", test.statusCode, w.Result().StatusCode)
			}
		})
	}
}
