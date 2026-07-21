package auth

import (
	"net/http"

	"github.com/JBK2116/vaulthook/internal/config"
)

// cookieSecure returns the Secure flag value based on the current environment.
// In production cookies are marked Secure; in development they are not,
// allowing local HTTP testing.
func cookieSecure() bool {
	return !config.Envs.IsDevelopment
}

// NewAccessCookie creates an http.Cookie for the access token with the
// provided value and max-age in seconds. Use a negative maxAge to expire
// the cookie immediately.
func NewAccessCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "access_token",
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: http.SameSiteLaxMode,
	}
}

// NewRefreshCookie creates an http.Cookie for the refresh token with the
// provided value and max-age in seconds. Use a negative maxAge to expire
// the cookie immediately.
func NewRefreshCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cookieSecure(),
		SameSite: http.SameSiteLaxMode,
	}
}

// ExpiredAccessCookie returns a cookie that immediately expires the
// access token in the browser.
func ExpiredAccessCookie() *http.Cookie {
	return NewAccessCookie("", -1)
}

// ExpiredRefreshCookie returns a cookie that immediately expires the
// refresh token in the browser.
func ExpiredRefreshCookie() *http.Cookie {
	return NewRefreshCookie("", -1)
}
