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

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

// Config manages all required tools and variables necessary for the application to run
// Serves as the singular object storing env variables, database connections and loggers
type Config struct {
	// db is a sql.DB connection object used to query and execute database related statements
	db *sql.DB
	// logger is a zerolog.Logger objected used to log statements during the applications runtime
	logger *zerolog.Logger
	// DB_TYPE is a string representing the type of database used in the application, e.g. "postgres"
	DB_TYPE string
	// DB_USER is a string representing the user connected to the database connection
	DB_USER string
	// DB_PASSWORD is a string representing the password to connect to the database connection
	DB_PASSWORD string
	// DB_HOST is a string representing the host connected to the database connection e.g. "localhost"
	DB_HOST string
	// DB_PORT is an int representing the host connected to the database connection e.g. 5432
	DB_PORT int
	// DB_NAME is a string representing the name of the database e.g. "vaulthook"
	DB_NAME string
	// IS_DEVELOPMENT is a boolean representing the environment that the application is running
	IS_DEVELOPMENT bool
}

// var Envs = initConfig()
//
// // initConfig loads all the required variables to configure the application
// func initConfig() Config {
//
// }

// getEnvString loads an environment variable using the provided name and returns the value in string format if found.
// If the environment variable is not found, panic is called.
func getEnvString(name string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		panic(fmt.Sprintf("Set the %s environment variable", name))
	}
	return value
}

// getEnvString loads an environment variable using the provided name and returns the value in boolean format if found.
// If the environment variable is not found, panic is called.
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

// getEnvString loads an environment variable using the provided name and returns the value in int format if found.
// If the environment variable is not found, panic is called.
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
