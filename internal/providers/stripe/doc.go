// Package stripe handles all business logic for inbound Stripe webhook processing.
//
// This package sits between the handler and the database, orchestrating
// signature verification, event persistence, and forwarding logic.
//
// # File Structure
//
//   - service.go - Business logic layer. Coordinates signature verification
//     via the shared event and provider repositories, forwarding header
//     preparation, and webhook persistence.
package stripe
