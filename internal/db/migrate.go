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
package db

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending database migrations from the
// migrations/ directory against the given database URL.
//
// Migrations are applied in order and are idempotent — running this
// function multiple times is safe. Already-applied migrations are
// skipped automatically.
func RunMigrations(databaseURL, migrationPath string) error {
	m, err := migrate.New(migrationPath, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialise migration runner: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			slog.Error("failed to close migration source", "error", sourceErr)
		}
		if dbErr != nil {
			slog.Error("failed to close migration database connection", "error", dbErr)
		}
	}()

	// Guard against active dirty schema state before executing
	version, dirty, err := m.Version()
	if err == nil && dirty {
		return fmt.Errorf(
			"database is in a dirty state at version %d: a previous migration did not complete cleanly. "+
				"Inspect the schema_migrations table and resolve manually",
			version,
		)
	}

	// Apply pending up-migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database migrations: no new migrations to apply", "current_version", version)
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Validate target state after successful application
	newVersion, _, err := m.Version()
	if err != nil {
		slog.Warn("could not read migration version after applying", "error", err)
		return nil
	}

	slog.Info("database migrations applied successfully", "version", newVersion)
	return nil
}
