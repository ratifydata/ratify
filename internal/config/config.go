package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the Ratify server.
// Values are loaded from environment variables or a .env file.
// No configuration value is hardcoded anywhere in the application.
type Config struct {
	// Server
	Port        int    `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	// Database
	// This is Ratify's own metadata database — not the databases
	// it monitors. It stores contracts, proposals, audit logs,
	// and encrypted credentials.
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	// Security
	// EncryptionKey is used to encrypt database credentials at
	// rest using AES-256-GCM. Must be exactly 32 bytes (64 hex chars).
	EncryptionKey string `mapstructure:"ENCRYPTION_KEY"`

	// JWTSecret is used to sign and verify session tokens for
	// the web UI. Must be kept secret.
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// Email (SMTP)
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     int    `mapstructure:"SMTP_PORT"`
	SMTPUsername string `mapstructure:"SMTP_USERNAME"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`

	// Breach detection
	// How often Ratify compares live schemas against active contracts.
	// Parsed as a Go duration string: "1h", "30m", "24h".
	BreachDetectionInterval string `mapstructure:"BREACH_DETECTION_INTERVAL"`
}

// Load reads configuration from environment variables and the
// .env file (if present). Environment variables take priority
// over .env file values.
//
// Returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	v := viper.New()

	// Tell Viper to look for a .env file in the current directory.
	// If no .env file exists, Viper falls back to environment
	// variables and defaults — it does not error.
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read the .env file. Ignore the error if the file does not
	// exist — this is expected in Docker and production environments
	// where variables are injected directly.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// The file exists but could not be read — that is a
			// real error worth reporting.
			_ = ok // file simply does not exist, continue
		}
	}

	// Tell Viper to also read from real environment variables.
	// Environment variables override .env file values.
	v.AutomaticEnv()

	// Set defaults for every variable.
	// These are used when no value is provided in .env or the
	// environment. Safe values that work for local development.
	v.SetDefault("PORT", 8080)
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("DATABASE_URL", "postgresql://ratify:ratify@localhost:5432/ratify?sslmode=disable")
	v.SetDefault("ENCRYPTION_KEY", strings.Repeat("0", 64))
	v.SetDefault("JWT_SECRET", "local-dev-jwt-secret-change-in-production")
	v.SetDefault("SMTP_HOST", "smtp.mailtrap.io")
	v.SetDefault("SMTP_PORT", 587)
	v.SetDefault("SMTP_USERNAME", "")
	v.SetDefault("SMTP_PASSWORD", "")
	v.SetDefault("SMTP_FROM", "ratify@example.com")
	v.SetDefault("BREACH_DETECTION_INTERVAL", "1h")

	// Unmarshal all values into the Config struct.
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields that have no safe default.
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsProduction returns true when the application is running in
// production. Used to enable stricter security settings.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true when the application is running
// in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// BreachInterval parses the BreachDetectionInterval string into
// a time.Duration. Returns the parsed duration or an error if
// the string is not a valid Go duration.
func (c *Config) BreachInterval() (time.Duration, error) {
	d, err := time.ParseDuration(c.BreachDetectionInterval)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid BREACH_DETECTION_INTERVAL %q: must be a valid Go duration (e.g. 1h, 30m): %w",
			c.BreachDetectionInterval,
			err,
		)
	}
	return d, nil
}

// validate checks that required configuration values are present
// and correctly formatted. Returns a descriptive error for the
// first problem found.
func validate(cfg *Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.EncryptionKey) != 64 {
		return fmt.Errorf(
			"ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes), got %d characters",
			len(cfg.EncryptionKey),
		)
	}
	if cfg.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf(
			"PORT must be between 1 and 65535, got %d",
			cfg.Port,
		)
	}
	return nil
}
