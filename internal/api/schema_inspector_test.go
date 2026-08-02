package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestSchemaConnectionHandlerSuccess(t *testing.T) {
	inspector := &stubInspectionValidator{}
	body := bytes.NewBufferString(`{
		"host":"database.example.com",
		"port":5432,
		"username":"ratify",
		"password":"secret",
		"database_name":"warehouse",
		"ssl_mode":"require",
		"ssl_enable":true,
		"driver_name":"pgx"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/schema/inspect", body)
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
		SSlEnable:    true,
		DriverName:   "pgx",
	}
	if inspector.params != want {
		t.Errorf("inspector params = %+v, want %+v", inspector.params, want)
	}
}

func TestSchemaConnectionHandlerInvalidJSON(t *testing.T) {
	inspector := &stubInspectionValidator{}
	req := httptest.NewRequest(http.MethodPost, "/schema/inspect", bytes.NewBufferString(`{`))
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
	req := httptest.NewRequest(http.MethodPost, "/schema/inspect", bytes.NewBufferString(`{"host":"database.example.com"}`))
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

func TestNewRouterProtectsSchemaInspectionWithAPIKey(t *testing.T) {
	router := NewRouter(nil)
	req := httptest.NewRequest(http.MethodPost, "/schema/inspect", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
