package stripe

import (
	"net/http/httptest"
	"testing"
)

func TestSetForwardHeaders(t *testing.T) {
	t.Run("forwards allowed headers only", func(t *testing.T) {
		headers := []byte(`{
			"Content-Type":     ["application/json"],
			"Stripe-Signature": ["t=123,v1=abc"],
			"User-Agent":       ["Stripe/1.0"],
			"Cache-Control":    ["no-cache"],
			"X-Forwarded-For":  ["10.0.0.1"],
			"Authorization":    ["Bearer secret"]
		}`)

		req := httptest.NewRequest("POST", "/", nil)
		err := SetForwardHeaders(req, headers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Allowed headers should be set.
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected Content-Type, got %q", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("Stripe-Signature") != "t=123,v1=abc" {
			t.Fatalf("expected Stripe-Signature, got %q", req.Header.Get("Stripe-Signature"))
		}
		if req.Header.Get("User-Agent") != "Stripe/1.0" {
			t.Fatalf("expected User-Agent, got %q", req.Header.Get("User-Agent"))
		}
		if req.Header.Get("Cache-Control") != "no-cache" {
			t.Fatalf("expected Cache-Control, got %q", req.Header.Get("Cache-Control"))
		}

		// Disallowed headers should NOT be forwarded.
		if req.Header.Get("X-Forwarded-For") != "" {
			t.Fatal("X-Forwarded-For should not be forwarded")
		}
		if req.Header.Get("Authorization") != "" {
			t.Fatal("Authorization should not be forwarded")
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		err := SetForwardHeaders(req, []byte(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		err := SetForwardHeaders(req, []byte(`not json`))
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("multiple values for same header", func(t *testing.T) {
		headers := []byte(`{"Content-Type":["application/json","text/plain"]}`)
		req := httptest.NewRequest("POST", "/", nil)
		err := SetForwardHeaders(req, headers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// http.Header.Get returns the first value.
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected first Content-Type value, got %q", req.Header.Get("Content-Type"))
		}
		// Both values should be present.
		vals := req.Header["Content-Type"]
		if len(vals) != 2 {
			t.Fatalf("expected 2 Content-Type values, got %d", len(vals))
		}
	})
}

func TestSetForwardHeaders_NilHeaders(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	err := SetForwardHeaders(req, nil)
	if err == nil {
		t.Fatal("expected error for nil headers")
	}
}

func TestSafePrefix(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"long string":   {input: "whsec_1234567890abcdef", expected: "whsec_"},
		"exactly 6":     {input: "123456", expected: "123456"},
		"shorter than 6": {input: "abc", expected: "abc"},
		"empty string":  {input: "", expected: ""},
		"single char":   {input: "x", expected: "x"},
		"exactly 7":     {input: "1234567", expected: "123456"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := safePrefix(tt.input)
			if got != tt.expected {
				t.Fatalf("safePrefix(%q): expected %q, got %q", tt.input, tt.expected, got)
			}
		})
	}
}

func TestSetForwardHeaders_NoAllowedHeaders(t *testing.T) {
	// Only disallowed headers are present — none should be forwarded.
	headers := []byte(`{"X-Custom": ["value"], "Authorization": ["Bearer tok"]}`)
	req := httptest.NewRequest("POST", "/", nil)
	err := SetForwardHeaders(req, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Header.Get("X-Custom") != "" {
		t.Fatal("X-Custom should not be forwarded")
	}
	if req.Header.Get("Authorization") != "" {
		t.Fatal("Authorization should not be forwarded")
	}
}

func TestSetForwardHeaders_EmptyHeaderValues(t *testing.T) {
	headers := []byte(`{"Stripe-Signature": []}`)
	req := httptest.NewRequest("POST", "/", nil)
	err := SetForwardHeaders(req, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty slice means no values were added.
	if req.Header.Get("Stripe-Signature") != "" {
		t.Fatal("expected no Stripe-Signature header")
	}
}

func TestSetForwardHeaders_MixedAllowedAndDisallowed(t *testing.T) {
	headers := []byte(`{
		"Content-Type": ["application/json"],
		"X-Forwarded-For": ["10.0.0.1"],
		"Stripe-Signature": ["t=1,v1=abc"],
		"Authorization": ["Bearer x"],
		"User-Agent": ["TestAgent/1.0"],
		"Cache-Control": ["no-cache"],
		"X-Custom": ["should-not-appear"]
	}`)
	req := httptest.NewRequest("POST", "/", nil)
	err := SetForwardHeaders(req, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Allowed headers present
	if req.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type missing")
	}
	if req.Header.Get("Stripe-Signature") != "t=1,v1=abc" {
		t.Error("Stripe-Signature missing")
	}
	if req.Header.Get("User-Agent") != "TestAgent/1.0" {
		t.Error("User-Agent missing")
	}
	if req.Header.Get("Cache-Control") != "no-cache" {
		t.Error("Cache-Control missing")
	}
	// Disallowed headers must NOT appear
	if req.Header.Get("X-Forwarded-For") != "" {
		t.Error("X-Forwarded-For should not be forwarded")
	}
	if req.Header.Get("Authorization") != "" {
		t.Error("Authorization should not be forwarded")
	}
	if req.Header.Get("X-Custom") != "" {
		t.Error("X-Custom should not be forwarded")
	}
}
