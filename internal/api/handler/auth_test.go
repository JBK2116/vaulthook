package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
)

func TestLoginHandler(t *testing.T) {
	tests := map[string]struct {
		input      []byte
		statusCode int
	}{
		// password field is improperly spelt "passwrd"
		"invalid body": {input: []byte(`{"email": "johndoe@gmail.com", "passwrd": "wrong password"}`), statusCode: http.StatusBadRequest},
		// email and password field is wrong here
		"invalid credentials": {input: []byte(`{"email": "johndoe@gmail.com", "password": "wrong password"}`), statusCode: http.StatusUnauthorized},
		// credentials are valid here
		"valid credentials": {input: getValidLoginCredentials(), statusCode: http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("POST", "http://localhost:8080/api/login", bytes.NewBuffer(test.input))
			w := httptest.NewRecorder()
			testAuthHandler.login(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}

}

func TestRefreshTokenHandler(t *testing.T) {
	tests := map[string]struct {
		token              func() string
		shouldIncludeToken bool
		statusCode         int
	}{
		"missing token in header": {token: func() string { return "" }, statusCode: http.StatusBadRequest, shouldIncludeToken: false},
		"expired token":           {token: func() string { return createExpiredRefreshToken(t) }, statusCode: http.StatusUnauthorized, shouldIncludeToken: true},
		"token invalid/missing":   {token: func() string { return "not a real token dub" }, statusCode: http.StatusUnauthorized, shouldIncludeToken: true},
		"valid token":             {token: func() string { return createRefreshToken(t) }, statusCode: http.StatusOK, shouldIncludeToken: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("POST", "http://localhost:8080/api/refresh", nil)
			if test.shouldIncludeToken {
				cookie := http.Cookie{
					Name:     "refresh_token",
					Value:    test.token(),
					MaxAge:   0, // cookie should always be included in test, in prod, expired cookies are handled with a simple if check
					HttpOnly: true,
					Secure:   !config.Envs.IsDevelopment,
					SameSite: http.SameSiteLaxMode,
					Path:     "/api/refresh",
				}
				r.AddCookie(&cookie)
			}
			w := httptest.NewRecorder()
			testAuthHandler.refreshToken(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}
}

func TestMeHandler(t *testing.T) {
	tests := map[string]struct {
		token              func() string
		shouldIncludeToken bool
		statusCode         int
	}{
		"missing token in header": {token: func() string { return "" }, shouldIncludeToken: false, statusCode: http.StatusUnauthorized},
		"expired token":           {token: func() string { return createExpiredAccessToken(t) }, shouldIncludeToken: true, statusCode: http.StatusUnauthorized},
		"valid token":             {token: func() string { return createAccessToken(t) }, shouldIncludeToken: true, statusCode: http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("POST", "http://localhost:8080/api/me", nil)
			if test.shouldIncludeToken {
				cookie := http.Cookie{
					Name:     "access_token",
					Value:    test.token(),
					MaxAge:   0, // cookie should always be included in test, in prod, expired cookies are handled with a simple if check
					HttpOnly: true,
					Secure:   !config.Envs.IsDevelopment,
					SameSite: http.SameSiteLaxMode,
				}
				r.AddCookie(&cookie)
			}
			w := httptest.NewRecorder()
			testAuthHandler.me(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}
}
