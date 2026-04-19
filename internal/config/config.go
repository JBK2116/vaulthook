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
	// DBType is the database driver type, e.g. "postgres".
	DBType string
	// DBUser is the database user.
	DBUser string
	// DBPassword is the database user's password.
	DBPassword string
	// DBHost is the database host, e.g. "localhost".
	DBHost string
	// DBPort is the database port, e.g. 5432.
	DBPort int
	// DBName is the name of the target database, e.g. "vaulthook".
	DBName string
	// UserEmail is the email of the authenticated application user.
	UserEmail string
	// UserPassword is the password of the authenticated application user.
	UserPassword string
	// LogLevel is the zerolog log level, e.g. 0 for debug.
	LogLevel int
	// JWTSecret is the HMAC secret used to sign and verify JWT tokens.
	JWTSecret string
	// AccessTokenTTL is the access token lifetime in minutes.
	AccessTokenTTL int
	// RefreshTokenTTL is the refresh token lifetime in hours.
	RefreshTokenTTL int
	// ThrottleMaxConcurrent is the maximum number of requests handled concurrently.
	ThrottleMaxConcurrent int
	// ThrottleMaxBacklog is the maximum number of requests queued while at capacity.
	ThrottleMaxBacklog int
	// ThrottleBacklogTimeout is the number of seconds a queued request may wait before timing out.
	ThrottleBacklogTimeout int
	// MaxRequestTime is the maximum number of seconds a request may run end-to-end.
	MaxRequestTime int
	// IsDevelopment indicates whether the application is running in a development environment.
	IsDevelopment bool
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
		DBType:                 getEnvString("DB_TYPE"),
		DBUser:                 getEnvString("DB_USER"),
		DBPassword:             getEnvString("DB_PASSWORD"),
		DBHost:                 getEnvString("DB_HOST"),
		DBPort:                 getEnvInt("DB_PORT"),
		DBName:                 getEnvString("DB_NAME"),
		UserEmail:              getEnvString("USER_EMAIL"),
		UserPassword:           getEnvString("USER_PASSWORD"),
		LogLevel:               getEnvInt("LOG_LEVEL"),
		JWTSecret:              getEnvString("TOKEN_SECRET"),
		AccessTokenTTL:         getEnvInt("ACCESS_TOKEN_TLL"),
		RefreshTokenTTL:        getEnvInt("REFRESH_TOKEN_TTL"),
		ThrottleMaxConcurrent:  getEnvInt("THROTTLE_MAX_CONCURRENT"),
		ThrottleMaxBacklog:     getEnvInt("THROTTLE_MAX_BACKLOG"),
		ThrottleBacklogTimeout: getEnvInt("THROTTLE_BACKLOG_TIMEOUT"),
		MaxRequestTime:         getEnvInt("MAX_REQUEST_TIME_LENGTH"),
		IsDevelopment:          getEnvBool("IS_DEVELOPMENT"),
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
