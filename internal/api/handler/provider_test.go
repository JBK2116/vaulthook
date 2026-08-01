package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/auth"
	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/go-chi/chi/v5"
)

func TestGetAll(t *testing.T) {
	tests := map[string]struct {
		accessToken        func() string
		shouldIncludeToken bool
		statusCode         int
	}{
		"missing access token": {accessToken: func() string { return "" }, shouldIncludeToken: false, statusCode: http.StatusUnauthorized},
		"invalid access token": {accessToken: func() string { return createExpiredAccessToken(t) }, shouldIncludeToken: true, statusCode: http.StatusUnauthorized},
		"valid access token":   {accessToken: func() string { return createAccessToken(t) }, shouldIncludeToken: true, statusCode: http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("GET", "http://localhost:8080/api/providers", nil)
			w := httptest.NewRecorder()
			accessT := test.accessToken()
			if test.shouldIncludeToken {
				cookie := http.Cookie{
					Name:     "access_token",
					Value:    accessT,
					MaxAge:   0,
					HttpOnly: true,
					Secure:   !config.Envs.IsDevelopment,
					SameSite: http.SameSiteLaxMode,
				}
				r.AddCookie(&cookie)
			}
			h := auth.Jwt(testAuthService)(http.HandlerFunc(testProviderHandler.getAll))
			h.ServeHTTP(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}
}

func TestConfigureHandler(t *testing.T) {
	tests := map[string]struct {
		accessToken        func() string
		providerName       string
		shouldIncludeToken bool
		statusCode         int
		body               []byte
	}{
		"missing access token":      {accessToken: func() string { return "" }, providerName: "Stripe", shouldIncludeToken: false, statusCode: http.StatusUnauthorized, body: []byte(`{}`)},
		"invalid access token":      {accessToken: func() string { return createExpiredAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusUnauthorized, body: []byte(`{}`)},
		"invalid body":              {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusBadRequest, body: []byte(`{"signing_secret": "jnjsd", "destination_ur": "https://collaboard.site/webhooks/stripe"}`)},
		"invalid retry count (low)": {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusBadRequest, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://example.com", "max_retries": -1, "max_req_second": 90}`)},
		"invalid retry count (high)": {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusBadRequest, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://example.com", "max_retries": 21, "max_req_second": 90}`)},
		"invalid req second (low)":  {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusBadRequest, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://example.com", "max_retries": 5, "max_req_second": -1}`)},
		"invalid req second (high)": {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusBadRequest, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://example.com", "max_retries": 5, "max_req_second": 1001}`)},
		"valid body stripe":         {accessToken: func() string { return createAccessToken(t) }, providerName: "Stripe", shouldIncludeToken: true, statusCode: http.StatusOK, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://collaboard.site/webhooks/stripe", "max_retries": 5, "max_req_second": 90}`)},
		"valid body github":         {accessToken: func() string { return createAccessToken(t) }, providerName: "Github", shouldIncludeToken: true, statusCode: http.StatusOK, body: []byte(`{"signing_secret": "jnjsd", "destination_url": "https://collaboard.site/webhooks/github", "max_retries": 5, "max_req_second": 100}`)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			id := getProviderID(ctx, t, test.providerName)
			accessT := test.accessToken()
			url := fmt.Sprintf("http://localhost:8080/api/providers/%s", id)
			r := httptest.NewRequest("PATCH", url, bytes.NewBuffer(test.body))
			// manually add the id into chi router context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", id)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()
			if test.shouldIncludeToken {
				cookie := http.Cookie{
					Name:     "access_token",
					Value:    accessT,
					MaxAge:   0,
					HttpOnly: true,
					Secure:   !config.Envs.IsDevelopment,
					SameSite: http.SameSiteLaxMode,
				}
				r.AddCookie(&cookie)
			}
			h := auth.Jwt(testAuthService)(http.HandlerFunc(testProviderHandler.configure))
			h.ServeHTTP(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}
}
