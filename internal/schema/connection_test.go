package schema

import (
	"context"
	"database/sql"
	"net/url"
	"os"
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
	testDB.Internal.Pool.Close()
	testutil.TerminateContainer(testDB.Internal.Container, testDB.External.Container)
	os.Exit(code)
}

func TestEstablishConnection(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		TestName string
		DSN      string
		Driver   string
		Want     bool
		WantErr  bool
	}{
		{
			TestName: "Connection_Success",
			DSN:      testDB.External.DSN,
			Driver:   "pgx",
			Want:     true,
			WantErr:  false,
		}, {
			TestName: "Unknown_Driver",
			DSN:      testDB.External.DSN,
			Driver:   "unknown-driver",
			WantErr:  true,
			Want:     false,
		}, {
			TestName: "Invalid_DSN",
			DSN:      testDB.External.DSN + "/invalid",
			Driver:   "pgx",
			WantErr:  true,
			Want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.TestName, func(t *testing.T) {
			db, err := EstablishConnection(ctx, tt.Driver, tt.DSN)
			if (err != nil) != tt.WantErr {
				t.Errorf("EstablishConnection() error = %v, WantErr %v", err, tt.WantErr)
			}

			if (db != nil) != tt.Want {
				t.Errorf("EstablishConnection() returned %v, want %v", (db != nil), tt.Want)
			}

		})
	}

}

func TestValidatePrivileges(t *testing.T) {
	t.Run("Privileges_Success", func(t *testing.T) {
		if err := ValidatePrivileges(t.Context(), testDB.External.DB); err != nil {
			t.Fatalf("ValidatePrivileges() error = %v, want nil", err)
		}
	})

	t.Run("Insufficient_Privileges", func(t *testing.T) {
		const (
			roleName = "schema_test_no_access"
			password = "schema-test-password"
		)

		if _, err := testDB.External.DB.ExecContext(t.Context(),
			`CREATE ROLE schema_test_no_access LOGIN PASSWORD 'schema-test-password'`); err != nil {
			t.Fatalf("create restricted role: %v", err)
		}
		t.Cleanup(func() {
			if _, err := testDB.External.DB.ExecContext(context.Background(),
				`DROP ROLE IF EXISTS schema_test_no_access`); err != nil {
				t.Errorf("drop restricted role: %v", err)
			}
		})

		restrictedDSN, err := url.Parse(testDB.External.DSN)
		if err != nil {
			t.Fatalf("parse external database DSN: %v", err)
		}
		restrictedDSN.User = url.UserPassword(roleName, password)

		restrictedDB, err := sql.Open("pgx", restrictedDSN.String())
		if err != nil {
			t.Fatalf("open restricted database connection: %v", err)
		}
		t.Cleanup(func() {
			if err := restrictedDB.Close(); err != nil {
				t.Errorf("close restricted database connection: %v", err)
			}
		})

		if err := ValidatePrivileges(t.Context(), restrictedDB); err == nil {
			t.Fatal("ValidatePrivileges() error = nil, want insufficient privileges error")
		}
	})
}

func TestValidatePrivilegesCanceledContext(t *testing.T) {
	db := testDB.External.DB
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ValidatePrivileges(ctx, db)
	if err == nil {
		t.Fatal("ValidatePrivileges() error = nil, want a context cancellation error")
	}
}
