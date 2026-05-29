package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ratifydata/ratify/internal/config"
	"github.com/ratifydata/ratify/internal/db"
)

func main() {
	// Load configuration from environment variables and .env file.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set up structured JSON logging.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Ratify server starting",
		"port", cfg.Port,
		"environment", cfg.Environment,
		"breach_interval", cfg.BreachDetectionInterval,
	)

	// Run database migrations before the server starts.
	// All pending migrations are applied in order. Already-applied
	// migrations are skipped. The server will not start if migrations fail.
	slog.Info("running database migrations")
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	// Placeholder — the real HTTP server starts in Card 15.
	fmt.Printf("Ratify server ready on port %d\n", cfg.Port)
}
