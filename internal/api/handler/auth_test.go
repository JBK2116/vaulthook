package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
			testHandler.login(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}

}

func TestRefreshTokenHandler(t *testing.T) {
	tests := map[string]struct {
		token      func() string
		statusCode int
	}{
		"missing token in header": {token: func() string { return "" }, statusCode: http.StatusBadRequest},
		"expired token":           {token: func() string { return createExpiredRefreshToken(t) }, statusCode: http.StatusUnauthorized},
		"token invalid/missing":   {token: func() string { return "not a real token dub" }, statusCode: http.StatusUnauthorized},
		"valid token":             {token: func() string { return createRefreshToken(t) }, statusCode: http.StatusOK},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			t.Cleanup(func() { afterEach(t) })
			r := httptest.NewRequest("POST", "http://localhost:8080/api/refresh", nil)
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", test.token()))
			w := httptest.NewRecorder()
			testHandler.refreshToken(w, r)
			res := w.Result()
			if res.StatusCode != test.statusCode {
				t.Fatalf("expected status code: %d, received: %d", test.statusCode, res.StatusCode)
			}
		})
	}
}
