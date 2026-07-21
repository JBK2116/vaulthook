package config

import (
	"os"
	"testing"
)

// TestInitConfig verifies that Init() correctly populates all fields of Envs
// from the .env file loaded during TestMain.
func TestInitConfig(t *testing.T) {
	// These values must match the .env file at the project root.
	if Envs.DBType != "postgres" {
		t.Fatalf("expected DBType 'postgres', got %q", Envs.DBType)
	}
	if Envs.DBUser != "postgres" {
		t.Fatalf("expected DBUser 'postgres', got %q", Envs.DBUser)
	}
	if Envs.DBHost != "localhost" {
		t.Fatalf("expected DBHost 'localhost', got %q", Envs.DBHost)
	}
	if Envs.DBPort != 5432 {
		t.Fatalf("expected DBPort 5432, got %d", Envs.DBPort)
	}
	if Envs.DBName != "vaulthook" {
		t.Fatalf("expected DBName 'vaulthook', got %q", Envs.DBName)
	}
	if Envs.IsDevelopment != true {
		t.Fatalf("expected IsDevelopment true, got %v", Envs.IsDevelopment)
	}
	if Envs.AccessTokenTTL != 10 {
		t.Fatalf("expected AccessTokenTTL 10, got %d", Envs.AccessTokenTTL)
	}
	if Envs.RefreshTokenTTL != 24 {
		t.Fatalf("expected RefreshTokenTTL 24, got %d", Envs.RefreshTokenTTL)
	}
	if Envs.MaxRetries != 5 {
		t.Fatalf("expected MaxRetries 5, got %d", Envs.MaxRetries)
	}
	if Envs.RetryIntervalSeconds != 15 {
		t.Fatalf("expected RetryIntervalSeconds 15, got %d", Envs.RetryIntervalSeconds)
	}
	if Envs.TotalQueueWorkers != 8 {
		t.Fatalf("expected TotalQueueWorkers 8, got %d", Envs.TotalQueueWorkers)
	}
	if Envs.TotalRetryWorkers != 12 {
		t.Fatalf("expected TotalRetryWorkers 12, got %d", Envs.TotalRetryWorkers)
	}
	if len(Envs.JWTSecret) < 10 {
		t.Fatalf("expected non-trivial JWTSecret, got length %d", len(Envs.JWTSecret))
	}
	if len(Envs.MasterKey) != 32 {
		t.Fatalf("expected MasterKey of 32 chars, got length %d", len(Envs.MasterKey))
	}
	if Envs.UserEmail == "" {
		t.Fatal("expected non-empty UserEmail")
	}
	if Envs.UserPassword == "" {
		t.Fatal("expected non-empty UserPassword")
	}
}

// TestGetEnvString_Success tests the helper with a set environment variable.
func TestGetEnvString_Success(t *testing.T) {
	key := "TEST_CONFIG_STRING_VAR"
	os.Setenv(key, "hello-world")
	defer os.Unsetenv(key)

	val := getEnvString(key)
	if val != "hello-world" {
		t.Fatalf("expected 'hello-world', got %q", val)
	}
}

// TestGetEnvString_Panics verifies getEnvString panics when the variable is unset.
func TestGetEnvString_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing env var, got none")
		}
	}()
	getEnvString("NONEXISTENT_VAR_FOR_TESTING_12345")
}

// TestGetEnvInt_Success tests the helper with a valid integer.
func TestGetEnvInt_Success(t *testing.T) {
	key := "TEST_CONFIG_INT_VAR"
	os.Setenv(key, "42")
	defer os.Unsetenv(key)

	val := getEnvInt(key)
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

// TestGetEnvInt_PanicsOnMissing verifies getEnvInt panics when the variable is unset.
func TestGetEnvInt_PanicsOnMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing env var, got none")
		}
	}()
	getEnvInt("NONEXISTENT_INT_VAR_FOR_TESTING_12345")
}

// TestGetEnvInt_PanicsOnInvalid verifies getEnvInt panics when the value is not an integer.
func TestGetEnvInt_PanicsOnInvalid(t *testing.T) {
	key := "TEST_CONFIG_INT_INVALID"
	os.Setenv(key, "not-a-number")
	defer os.Unsetenv(key)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid int env var, got none")
		}
	}()
	getEnvInt(key)
}

// TestGetEnvBool_Success tests the helper with valid boolean strings.
func TestGetEnvBool_Success(t *testing.T) {
	tests := map[string]bool{
		"true":  true,
		"false": false,
		"1":     true,
		"0":     false,
	}
	key := "TEST_CONFIG_BOOL_VAR"
	for input, expected := range tests {
		os.Setenv(key, input)
		val := getEnvBool(key)
		if val != expected {
			t.Fatalf("for input %q: expected %v, got %v", input, expected, val)
		}
	}
	os.Unsetenv(key)
}

// TestGetEnvBool_PanicsOnMissing verifies getEnvBool panics when the variable is unset.
func TestGetEnvBool_PanicsOnMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing env var, got none")
		}
	}()
	getEnvBool("NONEXISTENT_BOOL_VAR_FOR_TESTING_12345")
}

// TestGetEnvBool_PanicsOnInvalid verifies getEnvBool panics when the value is not a boolean.
func TestGetEnvBool_PanicsOnInvalid(t *testing.T) {
	key := "TEST_CONFIG_BOOL_INVALID"
	os.Setenv(key, "not-a-bool")
	defer os.Unsetenv(key)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid bool env var, got none")
		}
	}()
	getEnvBool(key)
}

// TestInitConfig_AllFieldsSet verifies that initConfig returns a Config with
// all required fields populated when all env vars are set.
func TestInitConfig_AllFieldsSet(t *testing.T) {
	// Save current env values we'll override.
	saved := saveEnv()
	defer restoreEnv(saved)

	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("USER_EMAIL", "test@test.com")
	os.Setenv("USER_PASSWORD", "testpass123")
	os.Setenv("LOG_LEVEL", "1")
	os.Setenv("TOKEN_SECRET", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	os.Setenv("ACCESS_TOKEN_TLL", "15")
	os.Setenv("REFRESH_TOKEN_TTL", "48")
	os.Setenv("THROTTLE_MAX_CONCURRENT", "200")
	os.Setenv("THROTTLE_MAX_BACKLOG", "100")
	os.Setenv("THROTTLE_BACKLOG_TIMEOUT", "20")
	os.Setenv("MAX_REQUEST_TIME_LENGTH", "120")
	os.Setenv("MAX_RETRIES", "3")
	os.Setenv("RETRY_INTERVAL_SECONDS", "30")
	os.Setenv("FORWARD_TIMEOUT_SECONDS", "10")
	os.Setenv("TOTAL_QUEUE_WORKERS", "4")
	os.Setenv("TOTAL_RETRY_WORKERS", "6")
	os.Setenv("MASTER_KEY", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4")
	os.Setenv("IS_DEVELOPMENT", "false")

	cfg := initConfig()

	if cfg.DBType != "postgres" {
		t.Errorf("DBType: expected postgres, got %q", cfg.DBType)
	}
	if cfg.DBPort != 5432 {
		t.Errorf("DBPort: expected 5432, got %d", cfg.DBPort)
	}
	if cfg.AccessTokenTTL != 15 {
		t.Errorf("AccessTokenTTL: expected 15, got %d", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 48 {
		t.Errorf("RefreshTokenTTL: expected 48, got %d", cfg.RefreshTokenTTL)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: expected 3, got %d", cfg.MaxRetries)
	}
	if cfg.IsDevelopment != false {
		t.Errorf("IsDevelopment: expected false, got %v", cfg.IsDevelopment)
	}
	if cfg.TotalQueueWorkers != 4 {
		t.Errorf("TotalQueueWorkers: expected 4, got %d", cfg.TotalQueueWorkers)
	}
	if cfg.TotalRetryWorkers != 6 {
		t.Errorf("TotalRetryWorkers: expected 6, got %d", cfg.TotalRetryWorkers)
	}
}

// TestConfig_StructFields test that all Config struct fields are addressable.
func TestConfig_StructFields(t *testing.T) {
	cfg := Config{}
	// Verify zero values are sensible.
	if cfg.DBPort != 0 {
		t.Error("expected zero-value DBPort to be 0")
	}
	if cfg.IsDevelopment != false {
		t.Error("expected zero-value IsDevelopment to be false")
	}
	if cfg.RetryIntervalSeconds != 0 {
		t.Error("expected zero-value RetryIntervalSeconds to be 0")
	}
}

// --- helpers ---

// saveEnv captures the current value of all env vars we may override in tests.
func saveEnv() map[string]string {
	vars := []string{
		"DB_TYPE", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
		"USER_EMAIL", "USER_PASSWORD", "LOG_LEVEL", "TOKEN_SECRET",
		"ACCESS_TOKEN_TLL", "REFRESH_TOKEN_TTL",
		"THROTTLE_MAX_CONCURRENT", "THROTTLE_MAX_BACKLOG", "THROTTLE_BACKLOG_TIMEOUT",
		"MAX_REQUEST_TIME_LENGTH", "MAX_RETRIES", "RETRY_INTERVAL_SECONDS",
		"FORWARD_TIMEOUT_SECONDS", "TOTAL_QUEUE_WORKERS", "TOTAL_RETRY_WORKERS",
		"MASTER_KEY", "IS_DEVELOPMENT",
	}
	saved := make(map[string]string, len(vars))
	for _, k := range vars {
		saved[k] = os.Getenv(k)
	}
	return saved
}

// restoreEnv restores environment variables from a previously saved map.
func restoreEnv(saved map[string]string) {
	for k, v := range saved {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}


// --- DB & Logger tests ---

// TestNewPG verifies that a database connection pool can be created
// and that it responds to pings.
func TestNewPG(t *testing.T) {
	ctx := t.Context()
	pg, err := NewPG(ctx)
	if err != nil {
		t.Fatalf("NewPG failed: %v", err)
	}
	if pg == nil {
		t.Fatal("expected non-nil postgres instance")
	}
	if pg.DB == nil {
		t.Fatal("expected non-nil DB pool")
	}
	// Verify the pool is alive.
	if err := pg.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

// TestNewPG_Singleton verifies that calling NewPG twice returns the same
// underlying pool (sync.Once semantics).
func TestNewPG_Singleton(t *testing.T) {
	ctx := t.Context()
	pg1, err := NewPG(ctx)
	if err != nil {
		t.Fatalf("first NewPG failed: %v", err)
	}
	pg2, err := NewPG(ctx)
	if err != nil {
		t.Fatalf("second NewPG failed: %v", err)
	}
	// Both should point to the same *postgres instance and same pool.
	if pg1 != pg2 {
		t.Fatal("expected NewPG to return the same singleton instance")
	}
	if pg1.DB != pg2.DB {
		t.Fatal("expected the same underlying pool")
	}
}

// TestNewLogger verifies that the package-level logger can be created.
func TestNewLogger(t *testing.T) {
	l, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TestNewLogger_Singleton verifies that NewLogger returns the same instance.
func TestNewLogger_Singleton(t *testing.T) {
	l1, err := NewLogger()
	if err != nil {
		t.Fatalf("first NewLogger failed: %v", err)
	}
	l2, err := NewLogger()
	if err != nil {
		t.Fatalf("second NewLogger failed: %v", err)
	}
	if l1 != l2 {
		t.Fatal("expected NewLogger to return the same singleton instance")
	}
}
