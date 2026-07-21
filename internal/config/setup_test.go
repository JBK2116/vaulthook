package config

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Load .env from project root so config.Init() can find all required vars.
	if err := godotenv.Load("../../.env"); err != nil {
		return
		// Fallback: .env may already be loaded by the test runner or system env.
		// We only panic if the required vars are truly missing.
	}
	Init()
	code := m.Run()
	os.Exit(code)
}
