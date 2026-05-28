package config

import (
	"os"
	"testing"
)

// clearConfigEnv unsets configuration environment variables for the scope of the test.
// It backs up original values and restores them afterward using t.Cleanup to prevent
// leaking side effects into downstream tests (e.g., breaking database connections).
func clearConfigEnv(t *testing.T) {
	t.Helper()
	vars := []string{
		"PORT", "ENVIRONMENT", "DATABASE_URL",
		"ENCRYPTION_KEY", "JWT_SECRET",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM",
		"BREACH_DETECTION_INTERVAL",
	}

	// 1. Capture and back up current environment states
	originalValues := make(map[string]string)
	for _, v := range vars {
		if val, exists := os.LookupEnv(v); exists {
			originalValues[v] = val
		}
	}

	// 2. Clear out the environment variables for this test context
	for _, v := range vars {
		if err := os.Unsetenv(v); err != nil {
			t.Fatalf("failed to unset %s: %v", v, err)
		}
	}

	// 3. Register a cleanup hook to seamlessly restore original state when this test exits
	t.Cleanup(func() {
		for _, v := range vars {
			if originalVal, wasSet := originalValues[v]; wasSet {
				_ = os.Setenv(v, originalVal)
			} else {
				_ = os.Unsetenv(v)
			}
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with defaults failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected default PORT 8080, got %d", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default ENVIRONMENT 'development', got %q", cfg.Environment)
	}
}

func TestLoad_EnvironmentVariableOverridesDefault(t *testing.T) {
	clearConfigEnv(t)

	if err := os.Setenv("PORT", "9090"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected PORT 9090 from environment, got %d", cfg.Port)
	}
}

func TestLoad_InvalidEncryptionKey(t *testing.T) {
	clearConfigEnv(t)

	if err := os.Setenv("ENCRYPTION_KEY", "tooshort"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short ENCRYPTION_KEY, got nil")
	}
}

func TestIsProduction(t *testing.T) {
	cfg := &Config{Environment: "production"}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction() to return true")
	}
}

func TestIsDevelopment(t *testing.T) {
	cfg := &Config{Environment: "development"}
	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment() to return true")
	}
}

func TestBreachInterval_Valid(t *testing.T) {
	cfg := &Config{BreachDetectionInterval: "1h"}
	d, err := cfg.BreachInterval()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Hours() != 1 {
		t.Errorf("expected 1 hour, got %v", d)
	}
}

func TestBreachInterval_Invalid(t *testing.T) {
	cfg := &Config{BreachDetectionInterval: "not-a-duration"}
	_, err := cfg.BreachInterval()
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}
