package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
)

func TestJwtMiddleware_NoCookie(t *testing.T) {
	svc := NewAuthService(
		config.Envs.JWTSecret,
		config.Envs.AccessTokenTTL,
		config.Envs.RefreshTokenTTL,
		nil,
		nil,
	)
	mw := Jwt(svc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing cookie, got %d", w.Result().StatusCode)
	}
}

func TestJwtMiddleware_ValidToken(t *testing.T) {
	svc := NewAuthService(
		config.Envs.JWTSecret,
		config.Envs.AccessTokenTTL,
		config.Envs.RefreshTokenTTL,
		nil,
		nil,
	)
	mw := Jwt(svc)

	now := time.Now()
	exp := now.Add(time.Duration(config.Envs.AccessTokenTTL) * time.Minute)
	token, err := svc.GenerateAccessToken(config.Envs.UserEmail, exp, now)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d", w.Result().StatusCode)
	}
}

func TestJwtMiddleware_ExpiredToken(t *testing.T) {
	svc := NewAuthService(
		config.Envs.JWTSecret,
		config.Envs.AccessTokenTTL,
		config.Envs.RefreshTokenTTL,
		nil,
		nil,
	)
	mw := Jwt(svc)

	now := time.Now()
	exp := now.Add(-1 * time.Minute) // already expired
	token, err := svc.GenerateAccessToken(config.Envs.UserEmail, exp, now)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: token,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Result().StatusCode)
	}
}

func TestJwtMiddleware_InvalidToken(t *testing.T) {
	svc := NewAuthService(
		config.Envs.JWTSecret,
		config.Envs.AccessTokenTTL,
		config.Envs.RefreshTokenTTL,
		nil,
		nil,
	)
	mw := Jwt(svc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "not.a.valid.jwt",
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", w.Result().StatusCode)
	}
}

func TestJwtMiddleware_TamperedToken(t *testing.T) {
	svc := NewAuthService(
		config.Envs.JWTSecret,
		config.Envs.AccessTokenTTL,
		config.Envs.RefreshTokenTTL,
		nil,
		nil,
	)
	mw := Jwt(svc)

	now := time.Now()
	exp := now.Add(time.Duration(config.Envs.AccessTokenTTL) * time.Minute)
	token, err := svc.GenerateAccessToken(config.Envs.UserEmail, exp, now)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Tamper with the token by flipping the last character.
	tampered := token[:len(token)-1] + "X"

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: tampered,
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for tampered token, got %d", w.Result().StatusCode)
	}
}
