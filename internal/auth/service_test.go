package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type loginBody struct {
	email    string
	password string
}

type loginResult struct {
	hasTokens bool
	err       error
}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		input  loginBody
		result loginResult
	}{
		"invalid credentials": {input: loginBody{email: "johndoe@gmail.com", password: "qwerty123"}, result: loginResult{hasTokens: false, err: ErrInvalidCredentials}},
		"valid credentials":   {input: loginBody{email: config.Envs.UserEmail, password: config.Envs.UserPassword}, result: loginResult{hasTokens: true, err: nil}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			ctx := context.Background()
			accessT, refreshT, err := testService.Login(ctx, test.input.email, test.input.password)
			if err != test.result.err {
				t.Fatal(err)
			}
			if test.result.hasTokens && (accessT == "" || refreshT == "") {
				t.Fatalf("expected tokens to be returned, received %s and %s", accessT, refreshT)
			}
			t.Cleanup(func() { afterEach(t) })
		})
	}
}

type tokenBody struct {
	setup func() string
}

type tokenResult struct {
	hasTokens bool
	err       error
}

func TestRefreshToken(t *testing.T) {
	tests := map[string]struct {
		input  tokenBody
		result tokenResult
	}{
		"expired token": {input: tokenBody{setup: func() string {
			token := createExpiredRefreshToken(t)
			return token
		}}, result: tokenResult{hasTokens: false, err: jwt.ErrTokenExpired}},
		"missing token": {input: tokenBody{setup: func() string { return "token not in database" }}, result: tokenResult{hasTokens: false, err: ErrTokenNotFound}},
		"valid token": {input: tokenBody{setup: func() string {
			token := createRefreshToken(t)
			return token
		}}, result: tokenResult{hasTokens: true, err: nil}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			ctx := context.Background()
			accessT, refreshT, err := testService.RefreshToken(ctx, test.input.setup())
			if !errors.Is(err, test.result.err) {
				t.Fatalf("expected error: %v, got error: %v", test.result.err, err)
			}
			if test.result.hasTokens && (accessT == "" || refreshT == "") {
				t.Fatalf("expected tokens to be returned, received %s and %s", accessT, refreshT)
			}
			t.Cleanup(func() { afterEach(t) })
		})
	}
}

func TestValidateAccessToken(t *testing.T) {
	tests := map[string]struct {
		input  tokenBody
		result tokenResult
	}{
		"expired token": {input: tokenBody{setup: func() string {
			token := createExpiredAccessToken(t)
			return token
		}}, result: tokenResult{err: jwt.ErrTokenExpired}},
		"invalid token": {input: tokenBody{setup: func() string { return "not.a.valid.token" }}, result: tokenResult{err: ErrInvalidToken}},
		"valid token": {input: tokenBody{setup: func() string {
			token := createRefreshToken(t)
			return token
		}}, result: tokenResult{err: nil}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			beforeEach(t)
			_, err := testService.ValidateAccessToken(test.input.setup())
			if err != test.result.err {
				t.Fatalf("expected error: %v, got error: %v", test.result.err, err)
			}
		})
		t.Cleanup(func() { afterEach(t) })
	}
}
