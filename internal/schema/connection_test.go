package schema

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ratifydata/ratify/internal/testutil"
)

var testDB *testutil.TestContainer

func TestMain(m *testing.M) {
	var err error
	testDB, err = testutil.InitializePostgresContainer()
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()
	testDB.Pool.Close()
	testutil.TerminateContainer(testDB.Container)
	os.Exit(code)
}

func TestEstablishConnection(t *testing.T) {
	db, err := EstablishConnection("pgx", testDB.Pool.Config().ConnString())
	if err != nil {
		t.Fatalf("EstablishConnection() error = %v, want nil", err)
	}
	if db == nil {
		t.Fatal("EstablishConnection() returned a nil database")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}
}

func TestEstablishConnectionUnknownDriver(t *testing.T) {
	db, err := EstablishConnection("unknown-schema-test-driver", "")
	if err == nil {
		t.Fatal("EstablishConnection() error = nil, want an unknown driver error")
	}
	if db != nil {
		t.Errorf("EstablishConnection() database = %v, want nil", db)
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("EstablishConnection() error = %q, want it to contain %q", err, "unknown driver")
	}
}

func TestEstablishConnectionInvalidDSN(t *testing.T) {
	db, err := EstablishConnection("pgx", "postgres://invalid:invalid@127.0.0.1:1/invalid")
	if err == nil {
		t.Fatal("EstablishConnection() error = nil, want a connection error")
	}
	if db != nil {
		t.Errorf("EstablishConnection() database = %v, want nil", db)
	}
}

func TestValidatePrivileges(t *testing.T) {
	db := openSQLDB(t, testDB.Pool.Config().ConnString())

	if err := ValidatePrivileges(context.Background(), db); err != nil {
		t.Fatalf("ValidatePrivileges() error = %v, want nil", err)
	}
}

func TestValidatePrivilegesInsufficientPrivileges(t *testing.T) {
	const (
		roleName = "schema_test_no_access"
		password = "schema-test-password"
	)

	_, err := testDB.Pool.Exec(context.Background(), "CREATE ROLE "+roleName+" LOGIN PASSWORD '"+password+"'")
	if err != nil {
		t.Fatalf("create restricted role: %v", err)
	}

	restrictedDSN, err := url.Parse(testDB.Pool.Config().ConnString())
	if err != nil {
		t.Fatalf("parse test database connection string: %v", err)
	}
	restrictedDSN.User = url.UserPassword(roleName, password)

	db := openSQLDB(t, restrictedDSN.String())
	err = ValidatePrivileges(context.Background(), db)
	if err == nil {
		t.Fatal("ValidatePrivileges() error = nil, want insufficient privileges")
	}
	if err.Error() != "logged in user has insufficient privileges" {
		t.Errorf("ValidatePrivileges() error = %q, want %q", err, "logged in user has insufficient privileges")
	}
}

func TestValidatePrivilegesCanceledContext(t *testing.T) {
	db := openSQLDB(t, testDB.Pool.Config().ConnString())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ValidatePrivileges(ctx, db)
	if err == nil {
		t.Fatal("ValidatePrivileges() error = nil, want a context cancellation error")
	}
}

func openSQLDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
	return db
}
