package testutil

import (
	"context"
	"database/sql"
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

type RatifyInternalTestContainer struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
}

type ClientExternalTestContainer struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	DSN       string
}
type TestContainer struct {
	Internal RatifyInternalTestContainer
	External ClientExternalTestContainer
}

// InitializePostgresContainer Pgx Pool for ratify internal data
func InitializePostgresContainer() (*TestContainer, error) {

	ctx := context.Background()

	//Initializes Internal Postgres Container
	internalPostgresContainer, err := initPostgresContainer(ctx, "ratify-test-database")
	if err != nil {
		slog.Error("error creating postgres container")
		TerminateContainer(internalPostgresContainer)
		return nil, err
	}

	connStr, err := internalPostgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		slog.Error("error getting postgres connection string")
		TerminateContainer(internalPostgresContainer)
		return nil, err
	}

	//todo: Find a way to retrieve the migration path directly
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsPath := filepath.Join(basepath, "..", "..", "migrations")
	//Use the pre-existing func for test containers
	if err := db.RunMigrations(connStr, fmt.Sprintf("file://%s", migrationsPath)); err != nil {
		slog.Error("error running migrations")
		TerminateContainer(internalPostgresContainer)
		return nil, err
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		slog.Error("error initializing postgres pool connection")
		TerminateContainer(internalPostgresContainer)
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		slog.Error("error pinging postgres pool")
		TerminateContainer(internalPostgresContainer)
		return nil, err
	}

	//Initializes External Client Postgres Container

	clientExtContainer, err := initPostgresContainer(ctx, "org-test-database")
	if err != nil {
		slog.Error("error initializing postgres container")
		TerminateContainer(clientExtContainer)
		return nil, err
	}
	connStr, err = clientExtContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		slog.Error("error getting postgres connection string")
		TerminateContainer(clientExtContainer)
		return nil, err
	}
	// Open database connection
	testDb, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("error initializing external postgres test database")
		TerminateContainer(clientExtContainer)
		return nil, err
	}
	//Create at least 2 tables
	query := `CREATE TABLE IF NOT EXISTS test_table_1 (
    id serial PRIMARY KEY,
    test_column VARCHAR(50) NOT NULL);
	
	CREATE TABLE IF NOT EXISTS test_table_2 (
    id serial PRIMARY KEY,
    test_column VARCHAR(50) NOT NULL);`

	_, err = testDb.ExecContext(ctx, query)
	if err != nil {
		slog.Error("error creating test table")
		TerminateContainer(clientExtContainer)
		return nil, err
	}

	slog.Info("postgres container initialized")
	return &TestContainer{
		Internal: RatifyInternalTestContainer{
			Container: internalPostgresContainer,
			Pool:      pool,
		},
		External: ClientExternalTestContainer{
			Container: clientExtContainer,
			DB:        testDb,
			DSN:       connStr,
		},
	}, nil

}

func initPostgresContainer(ctx context.Context, dbName string) (*postgres.PostgresContainer, error) {
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies())
	if err != nil {
		slog.Error("error creating postgres container")
		TerminateContainer(postgresContainer)
		return nil, err
	}
	return postgresContainer, nil

}

// TerminateContainer function to halt the container once called
func TerminateContainer(container ...*postgres.PostgresContainer) {
	defer func() {
		for _, container := range container {
			if err := container.Terminate(context.Background()); err != nil {
				slog.Error("error terminating postgres container")
			}
		}

	}()

}
