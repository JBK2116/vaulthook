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

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment variables required for the application to run.
type Config struct {
	// DB_TYPE is the database driver type, e.g. "postgres".
	DB_TYPE string
	// DB_USER is the database user.
	DB_USER string
	// DB_PASSWORD is the database user's password.
	DB_PASSWORD string
	// DB_HOST is the database host, e.g. "localhost".
	DB_HOST string
	// DB_PORT is the database port, e.g. 5432.
	DB_PORT int
	// DB_NAME is the name of the target database, e.g. "vaulthook".
	DB_NAME string
	// USER_EMAIL is the email of the authenticated application user.
	USER_EMAIL string
	// USER_PASSWORD is the password of the authenticated application user.
	USER_PASSWORD string
	// LOG_LEVEL is the zerolog log level, e.g. 0 for debug.
	LOG_LEVEL int
	// TOKEN_SECRET is the HMAC secret used to sign and verify JWT tokens.
	TOKEN_SECRET string
	// ACCESS_TOKEN_TLL is the access token lifetime in minutes.
	ACCESS_TOKEN_TLL int
	// REFRESH_TOKEN_TTL is the refresh token lifetime in hours.
	REFRESH_TOKEN_TTL int
	// THROTTLE_MAX_CONCURRENT is the maximum number of requests handled concurrently.
	THROTTLE_MAX_CONCURRENT int
	// THROTTLE_MAX_BACKLOG is the maximum number of requests queued while at capacity.
	THROTTLE_MAX_BACKLOG int
	// THROTTLE_BACKLOG_TIMEOUT is the number of seconds a queued request may wait before timing out.
	THROTTLE_BACKLOG_TIMEOUT int
	// MAX_REQUEST_TIME_LENGTH is the maximum number of seconds a request may run end-to-end.
	MAX_REQUEST_TIME_LENGTH int
	// IS_DEVELOPMENT indicates whether the application is running in a development environment.
	IS_DEVELOPMENT bool
}

// Envs is the package-level Config instance, initialized once at startup.
var Envs = initConfig()

// initConfig loads environment variables from .env and populates a Config.
// It panics if any required variable is missing or malformed.
func initConfig() Config {
	err := godotenv.Load()
	if err != nil {
		panic(fmt.Errorf("error loading environment variable: %w", err))
	}
	return Config{
		DB_TYPE:                  getEnvString("DB_TYPE"),
		DB_USER:                  getEnvString("DB_USER"),
		DB_PASSWORD:              getEnvString("DB_PASSWORD"),
		DB_HOST:                  getEnvString("DB_HOST"),
		DB_PORT:                  getEnvInt("DB_PORT"),
		DB_NAME:                  getEnvString("DB_NAME"),
		USER_EMAIL:               getEnvString("USER_EMAIL"),
		USER_PASSWORD:            getEnvString("USER_PASSWORD"),
		LOG_LEVEL:                getEnvInt("LOG_LEVEL"),
		TOKEN_SECRET:             getEnvString("TOKEN_SECRET"),
		ACCESS_TOKEN_TLL:         getEnvInt("ACCESS_TOKEN_TLL"),
		REFRESH_TOKEN_TTL:        getEnvInt("REFRESH_TOKEN_TTL"),
		THROTTLE_MAX_CONCURRENT:  getEnvInt("THROTTLE_MAX_CONCURRENT"),
		THROTTLE_MAX_BACKLOG:     getEnvInt("THROTTLE_MAX_BACKLOG"),
		THROTTLE_BACKLOG_TIMEOUT: getEnvInt("THROTTLE_BACKLOG_TIMEOUT"),
		MAX_REQUEST_TIME_LENGTH:  getEnvInt("MAX_REQUEST_TIME_LENGTH"),
		IS_DEVELOPMENT:           getEnvBool("IS_DEVELOPMENT"),
	}
}

// getEnvString returns the string value of the named environment variable.
// It panics if the variable is not set.
func getEnvString(name string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		panic(fmt.Sprintf("Set the %s environment variable", name))
	}
	return value
}

// getEnvBool returns the boolean value of the named environment variable.
// It panics if the variable is not set or cannot be parsed as a boolean.
func getEnvBool(name string) bool {
	value := os.Getenv(name)
	if len(value) == 0 {
		panic(fmt.Sprintf("Set the %s environment variable", name))
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("Environment variable %s must be a boolean, got: %s", name, value))
	}
	return boolValue
}

// getEnvInt returns the integer value of the named environment variable.
// It panics if the variable is not set or cannot be parsed as an integer.
func getEnvInt(name string) int {
	value := os.Getenv(name)
	if len(value) == 0 {
		panic(fmt.Sprintf("Set the %s environment variable", name))
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("Environment variable %s must be a int, got: %s", name, value))
	}
	return intValue
}
