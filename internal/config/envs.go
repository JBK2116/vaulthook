package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
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
	// RedisURL is the string url for the redis dependency.
	RedisURL string
	// UserEmail is the email of the authenticated application user.
	UserEmail string
	// UserPassword is the password of the authenticated application user.
	UserPassword string
	// LogLevel is the zerolog log level, e.g. 0 for debug.
	LogLevel zerolog.Level
	// JWTSecret is the HMAC secret used to sign and verify JWT tokens.
	JWTSecret string
	// MasterKey is the AES secret used to handle signing key encryption for providers.
	MasterKey string
	// IsDevelopment indicates whether the application is running in a development environment.
	IsDevelopment bool
}

//nolint:gochecknoglobals // Envs is the package-level Config instance, initialized once at startup.
var Envs Config

// initConfig loads environment variables from .env and populates a Config.
// It panics if any required variable is missing or malformed.
func initConfig() Config {
	return Config{
		DBType:        getEnvString("DB_TYPE"),
		DBUser:        getEnvString("DB_USER"),
		DBPassword:    getEnvString("DB_PASSWORD"),
		DBHost:        getEnvString("DB_HOST"),
		DBPort:        getEnvInt("DB_PORT"),
		DBName:        getEnvString("DB_NAME"),
		RedisURL:      getEnvString("REDIS_URL"),
		UserEmail:     getEnvString("USER_EMAIL"),
		UserPassword:  getEnvString("USER_PASSWORD"),
		LogLevel:      getEnvLevel("LOG_LEVEL"),
		JWTSecret:     getEnvString("TOKEN_SECRET"),
		MasterKey:     getEnvString("MASTER_KEY"),
		IsDevelopment: getEnvBool("IS_DEVELOPMENT"),
	}
}

// Init configures the `Envs` variable that stores all environment variables used in this project.
func Init() {
	Envs = initConfig()
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

// getEnvLevel returns the zerolog log level parsed from the named environment
// variable. It panics if the variable is not set or cannot be parsed as a
// valid zerolog level.
func getEnvLevel(name string) zerolog.Level {
	value := os.Getenv(name)
	if len(value) == 0 {
		panic(fmt.Sprintf("Set the %s environment variable", name))
	}
	level, err := zerolog.ParseLevel(value)
	if err != nil {
		panic(fmt.Sprintf("Environment variable %s must be a valid log level, got: %s", name, value))
	}
	return level
}
