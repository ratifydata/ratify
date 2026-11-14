// Copyright (C) 2026 Ratify Data
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func EstablishConnection(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	//Open returns a *DB struct validating the dns params passed.
	db, err := sql.Open(driver, dsn)
	if err != nil {
		slog.Error("Failed to open database connection")
		return nil, err
	}
	//We run a PingContext  to verify if we can reach the external database within 10 seconds.
	//PingContext preferred over Ping for a customized timeout
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		//Avoid leaking of the created db Connection
		_ = db.Close()
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
