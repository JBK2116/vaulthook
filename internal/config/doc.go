// Package config loads and validates all application-level configuration from environment variables.
//
// Configuration is initialized once at startup via a package-level variable.
// Any missing or malformed variable causes an immediate panic, preventing the
// application from starting in a misconfigured state.
//
// # Initialization Order
//
//  1. Load environment variables from .env via godotenv.
//
//  2. Validate and parse each required variable into its target type.
//
//  3. Return a populated Config struct assigned to the package-level Envs variable.
package config
