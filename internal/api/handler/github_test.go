package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/crypto"
	"github.com/JBK2116/vaulthook/internal/model"
)

const gitTestSecret = "github_test_secret_for_handler"

// insertGithubConfig sets up the GitHub provider with a test signing secret
// and destination URL so the handler can validate and insert webhooks.
func insertGithubConfig(ctx context.Context, t *testing.T, destURL string, secret string) {
	t.Helper()
	encrypted, err := crypto.EncryptSigningKey(secret)
	if err != nil {
		t.Fatalf("failed to encrypt github secret: %v", err)
	}
	_, err = testDB.Exec(ctx,
		`UPDATE providers SET signing_secret = $1, destination_url = $2, is_configured = true WHERE name = $3`,
		encrypted, destURL, model.Github)
	if err != nil {
		t.Fatalf("failed to configure github provider: %v", err)
	}
}

// computeGitHubSig generates a valid X-Hub-Signature-256 header value.
func computeGitHubSig(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestGithubReceive_MissingHookID(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	payload := []byte(`{"action":"opened"}`)
	r := httptest.NewRequest("POST", "http://localhost:8080/api/webhooks/github", bytes.NewBuffer(payload))
	r.Header.Set("Content-Type", "application/json")
	// Missing X-GitHub-Hook-ID → 400
	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestGithubReceive_MissingEvent(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	payload := []byte(`{"action":"opened"}`)
	r := httptest.NewRequest("POST", "http://localhost:8080/api/webhooks/github", bytes.NewBuffer(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Hook-ID", "12345")
	// Missing X-GitHub-Event → 400
	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Result().StatusCode)
	}
}

func TestGithubReceive_InvalidSignature(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	url := "http://localhost:8080/api/webhooks/github"
	insertGithubConfig(ctx, t, url, gitTestSecret)

	payload := []byte(`{"action":"opened"}`)
	sig := computeGitHubSig("wrong-secret", payload)

	r := httptest.NewRequest("POST", url, bytes.NewBuffer(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Hook-ID", "12345")
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", sig)

	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid signature, got %d", w.Result().StatusCode)
	}
}

func TestGithubReceive_InvalidPayload(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	url := "http://localhost:8080/api/webhooks/github"
	insertGithubConfig(ctx, t, url, gitTestSecret)

	// Valid signature but with a payload that is not valid JSON.
	// The DB insert will fail because payload is stored as JSONB.
	payload := []byte(`not json`)
	sig := computeGitHubSig(gitTestSecret, payload)

	r := httptest.NewRequest("POST", url, bytes.NewBuffer(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Hook-ID", "12345")
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", sig)

	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	// Invalid JSON causes a database error → 500.
	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 for invalid JSON payload, got %d", w.Result().StatusCode)
	}
}

func TestGithubReceive_Valid(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	url := "http://localhost:8080/api/webhooks/github"
	insertGithubConfig(ctx, t, url, gitTestSecret)

	payload := []byte(`{"action":"opened","ref":"refs/heads/main"}`)
	sig := computeGitHubSig(gitTestSecret, payload)

	r := httptest.NewRequest("POST", url, bytes.NewBuffer(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Hook-ID", "12345")
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", sig)

	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	// Verify the response is valid JSON with "queued" status.
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "queued" {
		t.Fatalf("expected status 'queued', got %q", body["status"])
	}
	if body["id"] != "12345" {
		t.Fatalf("expected id '12345', got %q", body["id"])
	}
}

func TestGithubReceive_BodyTooLarge(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	// Create a payload exceeding maxBodyBytes (25MB).
	largePayload := bytes.Repeat([]byte("x"), 25_000_001)
	sig := computeGitHubSig(gitTestSecret, largePayload)

	r := httptest.NewRequest("POST", "http://localhost:8080/api/webhooks/github", bytes.NewBuffer(largePayload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Hook-ID", "12345")
	r.Header.Set("X-GitHub-Event", "push")
	r.Header.Set("X-Hub-Signature-256", sig)

	w := httptest.NewRecorder()
	gitHandle.Receive(w, r)

	if w.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Result().StatusCode)
	}
}
