package middleware

import (
	"net/http"

	"github.com/JBK2116/vaulthook/internal"
	"github.com/JBK2116/vaulthook/internal/auth"
)

// Jwt returns a middleware that enforces Jwt authentication on protected routes.
//
// It extracts the bearer token from the Authorization header, validates it via
// the AuthService, and either passes the request to the next handler or responds
// with an appropriate status code.
func Jwt(s *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := internal.ExtractBearerToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_, err = s.ValidateAccessToken(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
