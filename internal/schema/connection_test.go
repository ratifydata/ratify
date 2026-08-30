package schema

import (
	"context"
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
	testDB.External.DB.Close()
	testutil.TerminateContainer(testDB.Internal.Container, testDB.External.Container)
	os.Exit(code)
}

func TestEstablishConnection(t *testing.T) {
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
			db, err := EstablishConnection(tt.Driver, tt.DSN)
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
	tests := []struct {
		TestName string
		SimErr   bool
		Want     bool
	}{
		{
			TestName: "Privileges_Success",
			Want:     false,
			SimErr:   false,
		}, {
			TestName: "InSufficient_Privileges",
			SimErr:   true,
			Want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.TestName, func(t *testing.T) {
			if tt.SimErr {
				sqlQuery := `CREATE ROLE schema_test_no_access LOGIN PASSWORD 'schema-test-password';
								SET ROLE schema_test_no_access;`
				//Creates a Role Not Granted Permission to Database And Switch's to that Role
				_, err := testDB.External.DB.ExecContext(context.Background(), sqlQuery)
				if err != nil {
					t.Fatalf("create restricted role: %v", err)
				}
			}

			err := ValidatePrivileges(context.Background(), testDB.External.DB)
			if (err != nil) != tt.Want {
				t.Errorf("ValidatePrivileges() error = %v, WantErr %v", err, tt.Want)
			}
		})
	}
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
