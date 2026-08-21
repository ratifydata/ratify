package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/config"
	"github.com/ratifydata/ratify/internal/schema"
)

type stubInspectionValidator struct {
	err    error
	calls  int
	params schema.ConnectionParams
}

func (f *stubInspectionValidator) SchemaInspection(_ context.Context, params schema.ConnectionParams) error {
	f.calls++
	f.params = params
	return f.err
}

type stubConnectionLister struct {
	err         error
	calls       int
	connections []schema.StoredConnection
}

type stubConnectionTester struct {
	err   error
	calls int
	id    pgtype.UUID
}

func (f *stubConnectionTester) TestConnection(_ context.Context, id pgtype.UUID) error {
	f.calls++
	f.id = id
	return f.err
}

func (f *stubConnectionLister) ListDatabaseConnections(_ context.Context) ([]schema.StoredConnection, error) {
	f.calls++
	return f.connections, f.err
}

func TestSchemaConnectionHandlerSuccess(t *testing.T) {
	inspector := &stubInspectionValidator{}
	body := bytes.NewBufferString(`{
		"host":"database.example.com",
		"port":5432,
		"username":"ratify",
		"password":"secret",
		"database_name":"warehouse",
		"ssl_mode":"require",
		"ssl_enabled":true,
		"driver_name":"pgx"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections", body)
	rec := httptest.NewRecorder()

	schemaConnectionHandler(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if inspector.calls != 1 {
		t.Fatalf("expected inspector to be called once, got %d calls", inspector.calls)
	}
	want := schema.ConnectionParams{
		Host:         "database.example.com",
		Port:         5432,
		Username:     "ratify",
		Password:     "secret",
		DatabaseName: "warehouse",
		SSLMode:      "require",
		SSlEnabled:   true,
		DriverName:   "pgx",
	}
	if inspector.params != want {
		t.Errorf("inspector params = %+v, want %+v", inspector.params, want)
	}
}

func TestSchemaConnectionHandlerInvalidJSON(t *testing.T) {
	inspector := &stubInspectionValidator{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	schemaConnectionHandler(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if inspector.calls != 0 {
		t.Fatalf("expected inspector not to be called, got %d calls", inspector.calls)
	}
}

func TestSchemaConnectionHandlerInspectionFailure(t *testing.T) {
	inspector := &stubInspectionValidator{err: errors.New("connection refused")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections", bytes.NewBufferString(`{"host":"database.example.com"}`))
	rec := httptest.NewRecorder()

	schemaConnectionHandler(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if inspector.calls != 1 {
		t.Fatalf("expected inspector to be called once, got %d calls", inspector.calls)
	}
	if rec.Body.String() != "connection refused\n" {
		t.Errorf("got response body %q, want %q", rec.Body.String(), "connection refused\\n")
	}
}

func TestListDatabaseConnectionsHandlerSuccess(t *testing.T) {
	id := pgtype.UUID{
		Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Valid: true,
	}
	want := []schema.StoredConnection{{
		ID:           id,
		DisplayName:  "Warehouse",
		Host:         "database.example.com",
		Port:         5432,
		DatabaseName: "analytics",
		Username:     "ratify",
		SSLEnabled:   true,
		SSLMode:      "require",
		Status:       "ACTIVE",
	}}
	lister := &stubConnectionLister{connections: want}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	rec := httptest.NewRecorder()

	listDatabaseConnections(lister).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if lister.calls != 1 {
		t.Fatalf("expected lister to be called once, got %d calls", lister.calls)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}

	var got []schema.StoredConnection
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("response connections = %+v, want %+v", got, want)
	}
}

func TestListDatabaseConnectionsHandlerEmpty(t *testing.T) {
	lister := &stubConnectionLister{connections: nil}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	rec := httptest.NewRecorder()

	listDatabaseConnections(lister).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "[]" {
		t.Errorf("response body = %q, want %q", rec.Body.String(), "[]")
	}
}

func TestListDatabaseConnectionsHandlerFailure(t *testing.T) {
	lister := &stubConnectionLister{err: errors.New("database unavailable")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	rec := httptest.NewRecorder()

	listDatabaseConnections(lister).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if lister.calls != 1 {
		t.Fatalf("expected lister to be called once, got %d calls", lister.calls)
	}
	if rec.Body.String() != "Internal Server Error\n" {
		t.Errorf("response body = %q, want %q", rec.Body.String(), "Internal Server Error\\n")
	}
}

func TestConnectionHandlerSuccess(t *testing.T) {
	inspector := &stubConnectionTester{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections/00000000-0000-0000-0000-000000000001/test", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	testConnection(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if inspector.calls != 1 {
		t.Fatalf("expected inspector to be called once, got %d calls", inspector.calls)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	if rec.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Errorf("got response body %q", rec.Body.String())
	}
}

func TestConnectionHandlerInspectorFailure(t *testing.T) {
	inspector := &stubConnectionTester{err: errors.New("connection refused")}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections/00000000-0000-0000-0000-000000000001/test", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	testConnection(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.String() != "{\"status\":\"err\",\"message\":\"connection refused\"}\n" {
		t.Errorf("got response body %q", rec.Body.String())
	}
}

func TestConnectionHandlerInvalidID(t *testing.T) {
	inspector := &stubConnectionTester{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections/not-a-uuid/test", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	testConnection(inspector).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if inspector.calls != 0 {
		t.Fatalf("expected inspector not to be called, got %d calls", inspector.calls)
	}
	if rec.Body.String() != "{\"status\":\"err\",\"message\":\"invalid connection id\"}\n" {
		t.Errorf("got response body %q", rec.Body.String())
	}
}

func TestNewRouterProtectsConnectionListingWithAPIKey(t *testing.T) {
	router := NewRouter(nil, &config.Config{EncryptionKey: "enc-test-key"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestNewRouterProtectsSchemaInspectionWithAPIKey(t *testing.T) {

	router := NewRouter(nil, &config.Config{EncryptionKey: "enc-test-key"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/connections", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
