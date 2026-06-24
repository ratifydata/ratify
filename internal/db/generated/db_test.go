package sqlc

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	tquery := New(testDB.Pool)

	if tquery == nil {
		t.Fatal("New() returned nil")
	}

	if tquery.db != testDB.Pool {
		t.Fatal("New() did not set db")
	}
}

func TestQueries_WithTx(t *testing.T) {
	ctx := context.Background()
	tx, err := testDB.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
	})

	tquery := New(testDB.Pool)
	txQuery := tquery.WithTx(tx)

	if txQuery == nil {
		t.Fatal("WithTx() returned nil")
	}

	if txQuery.db != tx {
		t.Fatal("WithTx() did not use supplied transaction")
	}

	if tquery.db != testDB.Pool {
		t.Fatal("WithTx() modified the original query instance")
	}
}
