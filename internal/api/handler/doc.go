// Package handler is responsible for creating all handlers used for corresponding provider endpoints
//
// This package serves as the highest level of code for all provider logic,
// therefore code in these handlers should be kept to a minimal.
// Leverage the code in each providers dedicated package to handle the main business logic for these endpoints.
//
// # Section 1: Example Handler Flow With Stripe
//
// 1. Receive the incoming http/https request.
//
// 2. Use the logic in `providers/stripe/` to respond accordingly to the request.
//
// 3. Send the result to it's appropriate destination.
package handler
