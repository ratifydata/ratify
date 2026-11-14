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
package sqlc

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	tquery := New(testDB.Internal.Pool)

	if tquery == nil {
		t.Fatal("New() returned nil")
	}

	if tquery.db != testDB.Internal.Pool {
		t.Fatal("New() did not set db")
	}
}

func TestQueries_WithTx(t *testing.T) {
	ctx := context.Background()
	tx, err := testDB.Internal.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
	})

	tquery := New(testDB.Internal.Pool)
	txQuery := tquery.WithTx(tx)

	if txQuery == nil {
		t.Fatal("WithTx() returned nil")
	}

	if txQuery.db != tx {
		t.Fatal("WithTx() did not use supplied transaction")
	}

	if tquery.db != testDB.Internal.Pool {
		t.Fatal("WithTx() modified the original query instance")
	}
}
