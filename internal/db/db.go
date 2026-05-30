package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a connection pool to the Ratify metadata database.
type Pool = pgxpool.Pool

// Connect creates a new connection pool to the PostgreSQL database
// at the given URL and verifies connectivity by pinging the server.
func Connect(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify the database is actually reachable. pgxpool.New does
	// not establish a connection immediately — it only parses the URL.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to reach database: %w", err)
	}

	return pool, nil
}
