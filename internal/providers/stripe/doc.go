// Package stripe handles all business logic for inbound Stripe webhook processing.
//
// This package sits between the handler and the database, orchestrating
// signature verification, event persistence, and forwarding logic.
//
// # File Structure
//
//   - repository.go - Database access layer for the webhook_events table.
//
//   - service.go - Business logic layer. Coordinates signature verification
//     via middleware and persistence via repository.go.
package stripe
