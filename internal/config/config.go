// Package config is responsible for initializing and exposing all application-level configuration.
//
// This package loads environment variables, and ensures that each variable is correctly configured.
// It exposes a single Config struct that is passed throughout the application.
//
// # Section 1: Initialization  Order
//
// 1. Load environment variables from .env.
//
// 2. Ensure that all necessary environment variables are properly configured.
//
// 3. Return a populated Config struct to the caller.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config manages all enviroment vairables necessary for the application to run
type Config struct {
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
	// LOG_LEVEL is an int representing the log level configured in the global logger e.g. 0
	LOG_LEVEL int
	// IS_DEVELOPMENT is a boolean representing the environment that the application is running
	IS_DEVELOPMENT bool
}

var Envs = initConfig()

// initConfig loads all the required variables to configure the application
func initConfig() Config {
	err := godotenv.Load()
	if err != nil {
		panic(fmt.Errorf("error loading environment variable: %w", err))
	}
	return Config{
		DB_TYPE:        getEnvString("DB_TYPE"),
		DB_USER:        getEnvString("DB_USER"),
		DB_PASSWORD:    getEnvString("DB_PASSWORD"),
		DB_HOST:        getEnvString("DB_HOST"),
		DB_PORT:        getEnvInt("DB_PORT"),
		DB_NAME:        getEnvString("DB_NAME"),
		LOG_LEVEL:      getEnvInt("LOG_LEVEL"),
		IS_DEVELOPMENT: getEnvBool("IS_DEVELOPMENT"),
	}
}

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
