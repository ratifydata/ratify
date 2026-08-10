package schema

import (
	"bytes"
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/ratifydata/ratify/internal/auth"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

const inspectionEncryptionKey = "0123456789abcdef0123456789abcdef"

func TestNewInspector(t *testing.T) {
	queries := sqlc.New(testDB.Pool)

	inspector := NewInspector(queries, inspectionEncryptionKey)

	if inspector == nil {
		t.Fatal("NewInspector() returned nil")
	}
	if inspector.db != queries {
		t.Error("NewInspector() did not retain the supplied queries")
	}
	if inspector.encKey != inspectionEncryptionKey {
		t.Errorf("NewInspector() encryption key = %q, want %q", inspector.encKey, inspectionEncryptionKey)
	}
}

func TestSchemaInspection(t *testing.T) {
	ctx := context.Background()
	queries := sqlc.New(testDB.Pool)
	org, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name: "Schema Inspection Test",
		Slug: "schema-inspection-test",
	})
	if err != nil {
		t.Fatalf("create test organization: %v", err)
	}

	params := testConnectionParams(t)
	params.SSlEnabled = false
	ctx = context.WithValue(ctx, "OrgID", org.ID)

	inspector := NewInspector(queries, inspectionEncryptionKey)
	if err := inspector.SchemaInspection(ctx, params); err != nil {
		t.Fatalf("SchemaInspection() error = %v, want nil", err)
	}

	connections, err := queries.ListDatabaseConnectionsByOrg(ctx, org.ID)
	if err != nil {
		t.Fatalf("list stored database connections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("stored connection count = %d, want 1", len(connections))
	}

	got := connections[0]
	if got.Host != params.Host || got.Port != int32(params.Port) ||
		got.DatabaseName != params.DatabaseName || got.Username != params.Username {
		t.Errorf("stored connection endpoint = %s:%d/%s as %s, want %s:%d/%s as %s",
			got.Host, got.Port, got.DatabaseName, got.Username,
			params.Host, params.Port, params.DatabaseName, params.Username)
	}
	if bytes.Equal(got.PasswordEncrypted, []byte(params.Password)) {
		t.Error("stored password was not encrypted")
	}
	plainText, err := auth.Decrypt(got.Nonce, got.PasswordEncrypted, []byte(inspectionEncryptionKey))
	if err != nil {
		t.Fatalf("decrypt stored password: %v", err)
	}
	if plainText != params.Password {
		t.Errorf("decrypted password = %q, want %q", plainText, params.Password)
	}
}

func TestSchemaInspectionConnectionFailure(t *testing.T) {
	inspector := NewInspector(sqlc.New(testDB.Pool), inspectionEncryptionKey)
	params := ConnectionParams{DriverName: "unknown-inspection-test-driver"}

	err := inspector.SchemaInspection(context.Background(), params)
	if err == nil {
		t.Fatal("SchemaInspection() error = nil, want an unknown driver error")
	}
	if !strings.Contains(err.Error(), "unknown driver") {
		t.Errorf("SchemaInspection() error = %q, want it to contain %q", err, "unknown driver")
	}
}

func TestSchemaInspectionEncryptionFailure(t *testing.T) {
	inspector := NewInspector(sqlc.New(testDB.Pool), "too-short")

	err := inspector.SchemaInspection(context.Background(), testConnectionParams(t))
	if err == nil {
		t.Fatal("SchemaInspection() error = nil, want an encryption key error")
	}
	if !strings.Contains(err.Error(), "secret must be 32 bytes") {
		t.Errorf("SchemaInspection() error = %q, want an encryption key error", err)
	}
}

func testConnectionParams(t *testing.T) ConnectionParams {
	t.Helper()

	dsn, err := url.Parse(testDB.Pool.Config().ConnString())
	if err != nil {
		t.Fatalf("parse test database connection string: %v", err)
	}
	port, err := strconv.Atoi(dsn.Port())
	if err != nil {
		t.Fatalf("parse test database port: %v", err)
	}
	password, _ := dsn.User.Password()

	return ConnectionParams{
		Host:         dsn.Hostname(),
		Port:         port,
		Username:     dsn.User.Username(),
		Password:     password,
		DatabaseName: strings.TrimPrefix(dsn.Path, "/"),
		SSLMode:      "disable",
		DriverName:   "pgx",
	}
}
