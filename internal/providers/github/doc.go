// Package github handles all business logic for inbound GitHub webhook processing.
//
// This package sits between the handler and the database, orchestrating
// signature verification, event persistence, and forwarding logic.
//
// # File Structure
//
//   - service.go - Business logic layer. Coordinates signature verification
//     via the shared event and provider repositories and webhook persistence.
package github
