package auth

import (
	"net/http"
	"testing"
)

func TestNewAccessCookie(t *testing.T) {
	t.Run("production", func(t *testing.T) {
		c := NewAccessCookie("token123", 600, true)
		if c.Name != "access_token" {
			t.Fatalf("expected name 'access_token', got %q", c.Name)
		}
		if c.Value != "token123" {
			t.Fatalf("expected value 'token123', got %q", c.Value)
		}
		if c.MaxAge != 600 {
			t.Fatalf("expected MaxAge 600, got %d", c.MaxAge)
		}
		if !c.HttpOnly {
			t.Fatal("expected HttpOnly true")
		}
		if !c.Secure {
			t.Fatal("expected Secure true in production")
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Fatalf("expected SameSite Lax, got %v", c.SameSite)
		}
	})

	t.Run("development", func(t *testing.T) {
		c := NewAccessCookie("token123", 600, false)
		if c.Secure {
			t.Fatal("expected Secure false in development")
		}
	})

	t.Run("expire", func(t *testing.T) {
		c := NewAccessCookie("", -1, true)
		if c.MaxAge != -1 {
			t.Fatalf("expected MaxAge -1 for expiry, got %d", c.MaxAge)
		}
		if c.Value != "" {
			t.Fatalf("expected empty value for expiry, got %q", c.Value)
		}
	})
}

func TestNewRefreshCookie(t *testing.T) {
	c := NewRefreshCookie("refresh123", 86400, true)
	if c.Name != "refresh_token" {
		t.Fatalf("expected name 'refresh_token', got %q", c.Name)
	}
	if c.Value != "refresh123" {
		t.Fatalf("expected value 'refresh123', got %q", c.Value)
	}
	if c.MaxAge != 86400 {
		t.Fatalf("expected MaxAge 86400, got %d", c.MaxAge)
	}
}

func TestExpiredAccessCookie(t *testing.T) {
	c := ExpiredAccessCookie(true)
	if c.MaxAge != -1 {
		t.Fatalf("expected MaxAge -1, got %d", c.MaxAge)
	}
	if c.Value != "" {
		t.Fatalf("expected empty value, got %q", c.Value)
	}
	if c.Name != "access_token" {
		t.Fatalf("expected name 'access_token', got %q", c.Name)
	}
}

func TestExpiredRefreshCookie(t *testing.T) {
	c := ExpiredRefreshCookie(false)
	if c.MaxAge != -1 {
		t.Fatalf("expected MaxAge -1, got %d", c.MaxAge)
	}
	if c.Value != "" {
		t.Fatalf("expected empty value, got %q", c.Value)
	}
	if c.Name != "refresh_token" {
		t.Fatalf("expected name 'refresh_token', got %q", c.Name)
	}
	if c.Secure {
		t.Fatal("expected Secure false in development")
	}
}
