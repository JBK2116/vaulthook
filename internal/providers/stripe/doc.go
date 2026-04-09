// Package stripe is responsible for all business logic related to Stripe webhook processing.
//
// This package sits between the handler and the database, handling signature verification,
// event persistence, and forwarding logic for all inbound Stripe webhook events.
//
// # Section 1: File Structure
//
// 1. repository.go - Database access layer. Handles all queries and statements against the webhook_events table.
//
// 2. service.go - Business logic layer. Orchestrates signature verification via auth/stripe_middleware.go and persistence via repository.go.
package stripe
