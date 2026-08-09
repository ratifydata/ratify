package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

func EstablishConnection(driver, dns string) (*sql.DB, error) {
	//Open returns a *DB struct validating the dns params passed.
	db, err := sql.Open(driver, dns)
	if err != nil {
		slog.Error("Failed to open database connection")
		return nil, err
	}
	//We run a PingContext  to verify if we can reach the external database within 10 seconds.
	//PingContext preferred over Ping for a customized timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("Failed to ping database. A network issue or wrong dns configuration")
		return nil, err
	}
	return db, nil
}

func ValidatePrivileges(ctx context.Context, db *sql.DB) error {
	const info_schema_query = `SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_type = 'BASE TABLE'
    AND table_schema NOT IN ('information_schema', 'pg_catalog')
    AND table_schema NOT LIKE 'pg_toast%'
	) AS has_accessible_tables;`

	var hasAccessibleTables bool
	err := db.QueryRowContext(ctx, info_schema_query).Scan(&hasAccessibleTables)
	if err != nil {
		slog.Error("information_schema query failed. A network issue or wrong dns configuration")
		return err
	}
	if !hasAccessibleTables {
		slog.Error("user does not have accessible base tables")
		return fmt.Errorf("logged in user has insufficient privileges")
	}
	return nil
}
