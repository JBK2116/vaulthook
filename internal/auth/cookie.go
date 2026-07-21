package auth

import "net/http"

// NewAccessCookie creates an http.Cookie for the access token with the
// provided value, max-age in seconds, and Secure flag. Use a negative
// maxAge to expire the cookie immediately.
func NewAccessCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "access_token",
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// NewRefreshCookie creates an http.Cookie for the refresh token with the
// provided value, max-age in seconds, and Secure flag. Use a negative
// maxAge to expire the cookie immediately.
func NewRefreshCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// ExpiredAccessCookie returns a cookie that immediately expires the
// access token in the browser.
func ExpiredAccessCookie(secure bool) *http.Cookie {
	return NewAccessCookie("", -1, secure)
}

// ExpiredRefreshCookie returns a cookie that immediately expires the
// refresh token in the browser.
func ExpiredRefreshCookie(secure bool) *http.Cookie {
	return NewRefreshCookie("", -1, secure)
}
