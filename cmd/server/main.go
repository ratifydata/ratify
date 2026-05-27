package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ratifydata/ratify/internal/config"
)

func main() {
	// Load configuration from environment variables and .env file.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Structured JSON logging — standard for production services.
	// In development, this is readable. In production, log
	// aggregation tools (Datadog, Loki) can parse it.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("Ratify server starting",
		"port", cfg.Port,
		"environment", cfg.Environment,
		"breach_interval", cfg.BreachDetectionInterval,
	)

	// Placeholder — the real server implementation would go here.
	fmt.Printf("Ratify server ready on port %d\n", cfg.Port)
}
