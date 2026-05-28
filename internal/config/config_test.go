package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Load with no .env file and no environment variables set.
	// Should succeed using defaults.
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
	if err := os.Setenv("PORT", "9090"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("PORT"); err != nil {
			t.Fatalf("failed to unset env var: %v", err)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("expected PORT 9090 from environment, got %d", cfg.Port)
	}
}

func TestLoad_InvalidEncryptionKey(t *testing.T) {
	if err := os.Setenv("ENCRYPTION_KEY", "tooshort"); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("ENCRYPTION_KEY"); err != nil {
			t.Fatalf("failed to unset env var: %v", err)
		}
	}()

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
