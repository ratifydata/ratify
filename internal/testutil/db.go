package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratifydata/ratify/internal/db"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type TestContainer struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
}

func InitializePostgresContainer() (*TestContainer, error) {

	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ratify-test-db"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies())
	if err != nil {
		slog.Error("error creating postgres container")
		TerminateContainer(postgresContainer)
		return nil, err
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		slog.Error("error getting postgres connection string")
		TerminateContainer(postgresContainer)
		return nil, err
	}

	//todo: Find a way to retrieve the migration path directly
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsPath := filepath.Join(basepath, "..", "..", "migrations")
	//Use the pre-existing func for test containers
	if err := db.RunMigrations(connStr, fmt.Sprintf("file://%s", migrationsPath)); err != nil {
		slog.Error("error running migrations")
		TerminateContainer(postgresContainer)
		return nil, err
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		slog.Error("error initializing postgres pool connection")
		TerminateContainer(postgresContainer)
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		slog.Error("error pinging postgres pool")
		TerminateContainer(postgresContainer)
		return nil, err
	}

	slog.Info("postgres container initialized")
	return &TestContainer{
		Container: postgresContainer,
		Pool:      pool,
	}, nil

}

// TerminateContainer function to halt the container once called
func TerminateContainer(container *postgres.PostgresContainer) {
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			slog.Error("error terminating postgres container")
		}
	}()

}
