package db

import (
	"testing"
)

func TestRunMigrations_InvalidURL(t *testing.T) {
	// Passing a clearly invalid URL should return an error from
	// the migration runner, not panic.
	err := RunMigrations("not-a-valid-database-url")
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
}
