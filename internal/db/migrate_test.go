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
	"testing"
)

func TestRunMigrations_InvalidURL(t *testing.T) {
	// Passing a clearly invalid URL should return an error from
	// the migration runner, not panic.
	err := RunMigrations("not-a-valid-database-url", "")
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
}
