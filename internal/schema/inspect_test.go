package schema

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/auth"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

const inspectionEncryptionKey = "0123456789abcdef0123456789abcdef"

const closeErrorDriverName = "inspect-close-error-driver"

func init() {
	sql.Register(closeErrorDriverName, closeErrorDriver{})
}

type fakeInspectorStore struct {
	create func(context.Context, sqlc.CreateDatabaseConnectionParams) (sqlc.DatabaseConnection, error)
	get    func(context.Context, pgtype.UUID) (sqlc.DatabaseConnection, error)
	list   func(context.Context, pgtype.UUID) ([]sqlc.ListDatabaseConnectionsByOrgRow, error)
	update func(context.Context, sqlc.UpdateDatabaseConnectionTestResultParams) error
}

func (f *fakeInspectorStore) CreateDatabaseConnection(ctx context.Context, params sqlc.CreateDatabaseConnectionParams) (sqlc.DatabaseConnection, error) {
	return f.create(ctx, params)
}

func (f *fakeInspectorStore) GetDatabaseConnection(ctx context.Context, id pgtype.UUID) (sqlc.DatabaseConnection, error) {
	return f.get(ctx, id)
}

func (f *fakeInspectorStore) ListDatabaseConnectionsByOrg(ctx context.Context, id pgtype.UUID) ([]sqlc.ListDatabaseConnectionsByOrgRow, error) {
	return f.list(ctx, id)
}

func (f *fakeInspectorStore) UpdateDatabaseConnectionTestResult(ctx context.Context, params sqlc.UpdateDatabaseConnectionTestResultParams) error {
	return f.update(ctx, params)
}

type closeErrorDriver struct{}

func (closeErrorDriver) Open(string) (driver.Conn, error) { return &closeErrorConn{}, nil }

type closeErrorConn struct{}

func (*closeErrorConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*closeErrorConn) Close() error               { return errors.New("close failed") }
func (*closeErrorConn) Begin() (driver.Tx, error)  { return nil, errors.New("not implemented") }
func (*closeErrorConn) Ping(context.Context) error { return nil }
func (*closeErrorConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &accessibleTableRows{}, nil
}

type accessibleTableRows struct {
	returned bool
}

func (*accessibleTableRows) Columns() []string { return []string{"has_accessible_tables"} }
func (*accessibleTableRows) Close() error      { return nil }
func (r *accessibleTableRows) Next(values []driver.Value) error {
	if r.returned {
		return io.EOF
	}
	r.returned = true
	values[0] = true
	return nil
}

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

	listed := connections[0]
	if listed.Host != params.Host || listed.Port != int32(params.Port) ||
		listed.DatabaseName != params.DatabaseName || listed.Username != params.Username {
		t.Errorf("stored connection endpoint = %s:%d/%s as %s, want %s:%d/%s as %s",
			listed.Host, listed.Port, listed.DatabaseName, listed.Username,
			params.Host, params.Port, params.DatabaseName, params.Username)
	}
	got, err := queries.GetDatabaseConnection(ctx, listed.ID)
	if err != nil {
		t.Fatalf("get stored database connection: %v", err)
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

func TestListDatabaseConnections(t *testing.T) {
	ctx := context.Background()
	queries := sqlc.New(testDB.Pool)
	org, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name: "List Connections Test",
		Slug: "list-connections-test",
	})
	if err != nil {
		t.Fatalf("create test organization: %v", err)
	}

	created, err := queries.CreateDatabaseConnection(ctx, sqlc.CreateDatabaseConnectionParams{
		OrgID:             org.ID,
		DisplayName:       "Warehouse",
		Host:              "database.example.com",
		Port:              5432,
		DatabaseName:      "analytics",
		Username:          "ratify",
		PasswordEncrypted: []byte("encrypted-password"),
		Nonce:             []byte("nonce"),
		SslEnabled:        true,
		SslMode:           "require",
		Status:            "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create test database connection: %v", err)
	}

	ctx = context.WithValue(ctx, "OrgID", org.ID)
	inspector := NewInspector(queries, inspectionEncryptionKey)
	connections, err := inspector.ListDatabaseConnections(ctx)
	if err != nil {
		t.Fatalf("ListDatabaseConnections() error = %v, want nil", err)
	}
	if len(connections) != 1 {
		t.Fatalf("ListDatabaseConnections() count = %d, want 1", len(connections))
	}

	want := StoredConnection{
		ID:           created.ID,
		DisplayName:  "Warehouse",
		Host:         "database.example.com",
		Port:         5432,
		DatabaseName: "analytics",
		Username:     "ratify",
		SSLEnabled:   true,
		SSLMode:      "require",
		Status:       "ACTIVE",
	}
	if connections[0] != want {
		t.Errorf("ListDatabaseConnections() = %+v, want %+v", connections[0], want)
	}
}

func TestListDatabaseConnectionsEmpty(t *testing.T) {
	ctx := context.Background()
	queries := sqlc.New(testDB.Pool)
	org, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name: "Empty Connections Test",
		Slug: "empty-connections-test",
	})
	if err != nil {
		t.Fatalf("create test organization: %v", err)
	}

	ctx = context.WithValue(ctx, "OrgID", org.ID)
	connections, err := NewInspector(queries, inspectionEncryptionKey).ListDatabaseConnections(ctx)
	if err != nil {
		t.Fatalf("ListDatabaseConnections() error = %v, want nil", err)
	}
	if connections == nil {
		t.Fatal("ListDatabaseConnections() returned nil, want an empty slice")
	}
	if len(connections) != 0 {
		t.Errorf("ListDatabaseConnections() count = %d, want 0", len(connections))
	}
}

func TestListDatabaseConnectionsMissingOrgID(t *testing.T) {
	inspector := NewInspector(sqlc.New(testDB.Pool), inspectionEncryptionKey)

	connections, err := inspector.ListDatabaseConnections(context.Background())
	if err == nil {
		t.Fatal("ListDatabaseConnections() error = nil, want missing OrgID error")
	}
	if connections != nil {
		t.Errorf("ListDatabaseConnections() = %+v, want nil", connections)
	}
	if !strings.Contains(err.Error(), "OrgID missing from context") {
		t.Errorf("ListDatabaseConnections() error = %q, want missing OrgID error", err)
	}
}

func TestConnection(t *testing.T) {
	ctx := context.Background()
	queries := sqlc.New(testDB.Pool)
	params := testConnectionParams(t)
	created := createStoredTestConnection(t, ctx, queries, params, inspectionEncryptionKey, "test-connection-success")

	inspector := NewInspector(queries, inspectionEncryptionKey)
	if err := inspector.TestConnection(ctx, created.ID); err != nil {
		t.Fatalf("TestConnection() error = %v, want nil", err)
	}

	got, err := queries.GetDatabaseConnection(ctx, created.ID)
	if err != nil {
		t.Fatalf("get tested database connection: %v", err)
	}
	if !got.LastTestedAt.Valid {
		t.Error("TestConnection() did not set last_tested_at")
	}
	if !got.LastTestPassed.Valid || !got.LastTestPassed.Bool {
		t.Errorf("TestConnection() last_test_passed = %+v, want true", got.LastTestPassed)
	}
}

func TestConnectionRecordsDecryptionFailure(t *testing.T) {
	ctx := context.Background()
	queries := sqlc.New(testDB.Pool)
	params := testConnectionParams(t)
	created := createStoredTestConnection(t, ctx, queries, params, inspectionEncryptionKey, "test-connection-decryption-failure")

	_, err := queries.UpdateDatabaseConnection(ctx, sqlc.UpdateDatabaseConnectionParams{
		ID:                created.ID,
		DisplayName:       created.DisplayName,
		Host:              created.Host,
		Port:              created.Port,
		DatabaseName:      created.DatabaseName,
		Username:          created.Username,
		PasswordEncrypted: []byte("invalid-ciphertext"),
		SslEnabled:        created.SslEnabled,
		SslMode:           created.SslMode,
		Status:            created.Status,
		Nonce:             []byte("invalidnonce"),
	})
	if err != nil {
		t.Fatalf("corrupt stored credentials: %v", err)
	}

	err = NewInspector(queries, inspectionEncryptionKey).TestConnection(ctx, created.ID)
	if err == nil {
		t.Fatal("TestConnection() error = nil, want a decryption error")
	}

	got, getErr := queries.GetDatabaseConnection(ctx, created.ID)
	if getErr != nil {
		t.Fatalf("get tested database connection: %v", getErr)
	}
	if !got.LastTestedAt.Valid {
		t.Error("TestConnection() did not set last_tested_at after failure")
	}
	if !got.LastTestPassed.Valid || got.LastTestPassed.Bool {
		t.Errorf("TestConnection() last_test_passed = %+v, want false", got.LastTestPassed)
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

func TestSchemaInspectionMissingOrgID(t *testing.T) {
	params := ConnectionParams{DriverName: closeErrorDriverName}
	inspector := NewInspector(&fakeInspectorStore{}, inspectionEncryptionKey)

	err := inspector.SchemaInspection(context.Background(), params)
	if err == nil || !strings.Contains(err.Error(), "OrgID missing from context") {
		t.Fatalf("SchemaInspection() error = %v, want missing OrgID error", err)
	}
}

func TestSchemaInspectionCreateFailure(t *testing.T) {
	wantErr := errors.New("create failed")
	store := &fakeInspectorStore{
		create: func(context.Context, sqlc.CreateDatabaseConnectionParams) (sqlc.DatabaseConnection, error) {
			return sqlc.DatabaseConnection{}, wantErr
		},
	}
	ctx := context.WithValue(context.Background(), "OrgID", pgtype.UUID{Valid: true})

	err := NewInspector(store, inspectionEncryptionKey).SchemaInspection(ctx, ConnectionParams{DriverName: closeErrorDriverName})
	if !errors.Is(err, wantErr) {
		t.Fatalf("SchemaInspection() error = %v, want %v", err, wantErr)
	}
}

func TestListDatabaseConnectionsQueryFailure(t *testing.T) {
	wantErr := errors.New("list failed")
	store := &fakeInspectorStore{
		list: func(context.Context, pgtype.UUID) ([]sqlc.ListDatabaseConnectionsByOrgRow, error) {
			return nil, wantErr
		},
	}
	ctx := context.WithValue(context.Background(), "OrgID", pgtype.UUID{Valid: true})

	connections, err := NewInspector(store, inspectionEncryptionKey).ListDatabaseConnections(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListDatabaseConnections() error = %v, want %v", err, wantErr)
	}
	if connections != nil {
		t.Errorf("ListDatabaseConnections() = %+v, want nil", connections)
	}
}

func TestConnectionGetFailure(t *testing.T) {
	wantErr := errors.New("get failed")
	store := &fakeInspectorStore{
		get: func(context.Context, pgtype.UUID) (sqlc.DatabaseConnection, error) {
			return sqlc.DatabaseConnection{}, wantErr
		},
	}

	err := NewInspector(store, inspectionEncryptionKey).TestConnection(context.Background(), pgtype.UUID{Valid: true})
	if !errors.Is(err, wantErr) {
		t.Fatalf("TestConnection() error = %v, want %v", err, wantErr)
	}
}

func TestConnectionRemoteFailure(t *testing.T) {
	encrypted, err := auth.Encrypt([]byte(inspectionEncryptionKey), "password")
	if err != nil {
		t.Fatalf("encrypt password: %v", err)
	}
	id := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	updated := false
	store := &fakeInspectorStore{
		get: func(context.Context, pgtype.UUID) (sqlc.DatabaseConnection, error) {
			return sqlc.DatabaseConnection{
				ID:                id,
				Host:              "127.0.0.1",
				Port:              1,
				Username:          "invalid",
				DatabaseName:      "invalid",
				SslMode:           "disable",
				PasswordEncrypted: encrypted.CipherText,
				Nonce:             encrypted.Nonce,
			}, nil
		},
		update: func(_ context.Context, params sqlc.UpdateDatabaseConnectionTestResultParams) error {
			updated = true
			if !params.LastTestPassed.Valid || params.LastTestPassed.Bool {
				t.Errorf("last test passed = %+v, want false", params.LastTestPassed)
			}
			return nil
		},
	}

	err = NewInspector(store, inspectionEncryptionKey).TestConnection(context.Background(), id)
	if err == nil {
		t.Fatal("TestConnection() error = nil, want remote connection error")
	}
	if !updated {
		t.Fatal("TestConnection() did not persist the failed result")
	}
}

func TestConnectionJoinsResultUpdateFailure(t *testing.T) {
	decryptErrText := "cipher: message authentication failed"
	updateErr := errors.New("update failed")
	id := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	store := &fakeInspectorStore{
		get: func(context.Context, pgtype.UUID) (sqlc.DatabaseConnection, error) {
			return sqlc.DatabaseConnection{
				ID:                id,
				PasswordEncrypted: []byte("invalid ciphertext"),
				Nonce:             make([]byte, 12),
			}, nil
		},
		update: func(context.Context, sqlc.UpdateDatabaseConnectionTestResultParams) error {
			return updateErr
		},
	}

	err := NewInspector(store, inspectionEncryptionKey).TestConnection(context.Background(), id)
	if err == nil {
		t.Fatal("TestConnection() error = nil, want joined error")
	}
	if !strings.Contains(err.Error(), decryptErrText) {
		t.Errorf("TestConnection() error = %q, want decryption error", err)
	}
	if !errors.Is(err, updateErr) {
		t.Errorf("TestConnection() error = %v, want joined update error %v", err, updateErr)
	}
}

func TestEstablishRemoteConnectionValidationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := establishRemoteConnection(ctx, ConnectionParams{DriverName: closeErrorDriverName})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("establishRemoteConnection() error = %v, want context cancellation", err)
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

func createStoredTestConnection(
	t *testing.T,
	ctx context.Context,
	queries *sqlc.Queries,
	params ConnectionParams,
	encryptionKey string,
	slug string,
) sqlc.DatabaseConnection {
	t.Helper()

	org, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name: slug,
		Slug: slug,
	})
	if err != nil {
		t.Fatalf("create test organization: %v", err)
	}
	encrypted, err := auth.Encrypt([]byte(encryptionKey), params.Password)
	if err != nil {
		t.Fatalf("encrypt test connection password: %v", err)
	}
	created, err := queries.CreateDatabaseConnection(ctx, sqlc.CreateDatabaseConnectionParams{
		OrgID:             org.ID,
		DisplayName:       slug,
		Host:              params.Host,
		Port:              int32(params.Port),
		DatabaseName:      params.DatabaseName,
		Username:          params.Username,
		PasswordEncrypted: encrypted.CipherText,
		SslEnabled:        params.SSlEnabled,
		SslMode:           params.SSLMode,
		Status:            "ACTIVE",
		Nonce:             encrypted.Nonce,
	})
	if err != nil {
		t.Fatalf("create stored database connection: %v", err)
	}
	return created
}
