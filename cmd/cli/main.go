package main

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"

	"github.com/ratifydata/ratify/internal/cli"
	"github.com/ratifydata/ratify/internal/config"
	"github.com/ratifydata/ratify/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if err = validateKey(cfg.EncryptionKey); err != nil {
		slog.Error("failed to validate key")
		os.Exit(1)
	}
	//Initialize DB setup
	// Establish concurrent pool configuration
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	cli.Execute(cfg, pool)
}

func validateKey(encKey string) error {
	if encKey == "" {
		return errors.New("key cannot be empty")
	}

	decoded, err := hex.DecodeString(encKey)
	if err != nil {
		slog.Error("failed to decode key")
		return err
	}

	if len(decoded) != 32 {
		slog.Error("invalid key length")
		return errors.New("invalid key length")
	}
	return nil
}
