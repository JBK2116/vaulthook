package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestLogoutHandler(t *testing.T) {
	tests := map[string]struct {
		refreshToken       func() string
		shouldIncludeToken bool
		statusCode         int
	}{
		"missing token":     {refreshToken: func() string { return "" }, shouldIncludeToken: false, statusCode: http.StatusBadRequest},
		"valid token":       {refreshToken: func() string { return createRefreshToken(t) }, shouldIncludeToken: true, statusCode: http.StatusOK},
		"nonexistent token": {refreshToken: func() string { return "nonexistent_token_string" }, shouldIncludeToken: true, statusCode: http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("POST", "http://localhost:8080/api/logout", nil)
			if test.shouldIncludeToken {
				cookie := http.Cookie{
					Name:     "refresh_token",
					Value:    test.refreshToken(),
					HttpOnly: true,
					Secure:   !config.Envs.IsDevelopment,
					SameSite: http.SameSiteLaxMode,
				}
				r.AddCookie(&cookie)
			}
			w := httptest.NewRecorder()
			testAuthHandler.logout(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status %d, got %d", test.statusCode, res.StatusCode)
			}
		})
	}
}

func TestEventsGetAll(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	// Insert a test webhook via the event repo so there's data.
	ctx := context.Background()
	params := model.CreateWebhookParams{
		ProviderID:  getProviderUUID(ctx, t, "Stripe"),
		Provider:    "Stripe",
		EventType:   "test.event",
		Headers:     json.RawMessage(`{}`),
		Payload:     json.RawMessage(`{}`),
		ForwardedTo: "https://example.com",
		ReceivedAt:  time.Now().UTC(),
	}
	if _, err := eventRepo.InsertWebhook(ctx, params); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	r := httptest.NewRequest("GET", "http://localhost:8080/api/events", nil)
	accessT := createAccessToken(t)
	r.AddCookie(&http.Cookie{
		Name: "access_token", Value: accessT,
		HttpOnly: true, Secure: !config.Envs.IsDevelopment, SameSite: http.SameSiteLaxMode,
	})
	w := httptest.NewRecorder()

	h := auth.Jwt(testAuthService)(http.HandlerFunc(eventsHandler.getAll))
	h.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestEventsGetStats(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	r := httptest.NewRequest("GET", "http://localhost:8080/api/events/stats", nil)
	accessT := createAccessToken(t)
	r.AddCookie(&http.Cookie{
		Name: "access_token", Value: accessT,
		HttpOnly: true, Secure: !config.Envs.IsDevelopment, SameSite: http.SameSiteLaxMode,
	})
	w := httptest.NewRecorder()

	h := auth.Jwt(testAuthService)(http.HandlerFunc(eventsHandler.getStats))
	h.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestEventsReplayEvent(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	ctx := context.Background()
	params := model.CreateWebhookParams{
		ProviderID:  getProviderUUID(ctx, t, "Stripe"),
		Provider:    "Stripe",
		EventType:   "test.event",
		Headers:     json.RawMessage(`{}`),
		Payload:     json.RawMessage(`{}`),
		ForwardedTo: "https://example.com",
		ReceivedAt:  time.Now().UTC(),
	}
	hook, err := eventRepo.InsertWebhook(ctx, params)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	r := httptest.NewRequest("POST", "http://localhost:8080/api/events/"+hook.ID.String()+"/replay", nil)
	accessT := createAccessToken(t)
	r.AddCookie(&http.Cookie{
		Name: "access_token", Value: accessT,
		HttpOnly: true, Secure: !config.Envs.IsDevelopment, SameSite: http.SameSiteLaxMode,
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", hook.ID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h := auth.Jwt(testAuthService)(http.HandlerFunc(eventsHandler.replayEvent))
	h.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestEventsReplayEvent_InvalidUUID(t *testing.T) {
	beforeEach(t)
	t.Cleanup(func() { afterEach(t) })

	r := httptest.NewRequest("POST", "http://localhost:8080/api/events/not-a-uuid/replay", nil)
	accessT := createAccessToken(t)
	r.AddCookie(&http.Cookie{
		Name: "access_token", Value: accessT,
		HttpOnly: true, Secure: !config.Envs.IsDevelopment, SameSite: http.SameSiteLaxMode,
	})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h := auth.Jwt(testAuthService)(http.HandlerFunc(eventsHandler.replayEvent))
	h.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", res.StatusCode)
	}
}

// helpers

func getProviderUUID(ctx context.Context, t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx, `SELECT id FROM providers WHERE name = $1`, name).Scan(&id)
	if err != nil {
		t.Fatalf("provider %q not found: %v", name, err)
	}
	return id
}
