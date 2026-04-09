// Package router is responsible for registering all handlers to their corresponding routes
//
// This package serves as the orchestrator for all routes in this application.
// Each route attaches a function defined in the `handler` package.
//
// # Section 1: Example Router Flow With Stripe
//
// 1. Register the route `/api/webhooks/stripe`.
//
// 2. Import the corresponding handler from the `handler package`.
//
// 3. Attach the handler to the route.
package router
