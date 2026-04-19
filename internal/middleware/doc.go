// Package middleware provides HTTP middleware for provider-specific and authentication request handling.
//
// Each provider has a dedicated middleware file following the naming convention
// providername_middleware.go (e.g. stripe_middleware.go). Provider middleware
// is responsible for any pre-processing a provider requires before the request
// reaches its handler, such as signature verification and payload validation.
package middleware
