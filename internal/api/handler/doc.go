// Package handler provides HTTP handlers for all provider and authentication endpoints.
//
// This package is the topmost layer of the provider request lifecycle.
// Handler logic should remain thin, request decoding, response writing,
// and error translation only. All business logic must be delegated to the
// corresponding provider package.
//
// # Handler Flow (Stripe Example)
//
//  1. Receive the incoming HTTP request.
//
//  2. Delegate to the appropriate logic in providers/stripe/.
//
//  3. Write the result back to the response writer.
package handler
