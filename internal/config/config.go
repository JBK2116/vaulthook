// Package config is responsible for initializing and exposing all application-level configuration.
//
// This package loads environment variables, initializes the database connection,
// and sets up the logger. It exposes a single Config struct that is passed
// throughout the application.
//
// # Section 1: Initialization  Order
//
// 1. Load environment variables from .env.
// 2. Initialize the logger.
// 3. Open and verify the database connection.
// 4. Return a populated Config struct to the caller.
package config
