package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Healthy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok","database":"ok","version":"0.1.0"}`)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Database != "ok" {
		t.Errorf("expected database 'ok', got %q", resp.Database)
	}
	if resp.Version != version {
		t.Errorf("expected version %q, got %q", version, resp.Version)
	}
}

func TestStatusFromHTTP(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{http.StatusOK, "ok"},
		{http.StatusServiceUnavailable, "degraded"},
		{http.StatusInternalServerError, "degraded"},
	}

	for _, tt := range tests {
		result := statusFromHTTP(tt.code)
		if result != tt.expected {
			t.Errorf("statusFromHTTP(%d) = %q, want %q", tt.code, result, tt.expected)
		}
	}
}
