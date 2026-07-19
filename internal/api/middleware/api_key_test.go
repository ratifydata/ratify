package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type testAPIKeyAuthenticator struct {
	apiKey    *sqlc.ApiKey
	authErr   error
	updateErr error

	authCalls   int
	updateCalls int
	prefix      string
	keyHash     string
	updatedID   pgtype.UUID
	updatedCtx  context.Context
	updated     chan struct{}
}

func (f *testAPIKeyAuthenticator) ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error) {
	f.authCalls++
	f.prefix = prefix
	f.keyHash = keyHash

	if f.authErr != nil {
		return nil, f.authErr
	}

	return f.apiKey, nil
}

func (f *testAPIKeyAuthenticator) UpdateVerificationTimestamp(ctx context.Context, id pgtype.UUID) error {
	f.updateCalls++
	f.updatedID = id
	f.updatedCtx = ctx
	if f.updated != nil {
		close(f.updated)
	}
	return f.updateErr
}

func TestAuthHandler_AuthorizationHeaderMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	authenticator := &testAPIKeyAuthenticator{}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := authHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.authCalls != 0 {
		t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.authCalls)
	}
	if authenticator.updateCalls != 0 {
		t.Fatalf("expected verification timestamp not to be updated, got %d calls", authenticator.updateCalls)
	}
}

func TestAuthHandler_InvalidAuthorizationScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic abcdefgh.secret")
	rec := httptest.NewRecorder()
	authenticator := &testAPIKeyAuthenticator{}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.authCalls != 0 {
		t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.authCalls)
	}
	if authenticator.updateCalls != 0 {
		t.Fatalf("expected verification timestamp not to be updated, got %d calls", authenticator.updateCalls)
	}
}

func TestAuthHandler_MalformedAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{
			name:          "short api key",
			authorization: "Bearer short",
		},
		{
			name:          "missing prefix separator",
			authorization: "Bearer abcdefghsecret",
		},
		{
			name:          "empty secret",
			authorization: "Bearer abcdefgh.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.authorization)
			rec := httptest.NewRecorder()
			authenticator := &testAPIKeyAuthenticator{}
			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := authHandler(authenticator)(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if nextCalled {
				t.Fatal("expected next handler not to be called")
			}
			if authenticator.authCalls != 0 {
				t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.authCalls)
			}
			if authenticator.updateCalls != 0 {
				t.Fatalf("expected verification timestamp not to be updated, got %d calls", authenticator.updateCalls)
			}
		})
	}
}

func TestAuthHandler_AuthenticationFailed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer abcdefgh.secret")
	rec := httptest.NewRecorder()
	authenticator := &testAPIKeyAuthenticator{authErr: errors.New("invalid api key")}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.authCalls != 1 {
		t.Fatalf("expected authenticator to be called once, got %d calls", authenticator.authCalls)
	}
	if authenticator.updateCalls != 0 {
		t.Fatalf("expected verification timestamp not to be updated, got %d calls", authenticator.updateCalls)
	}
	if authenticator.prefix != "abcdefgh" {
		t.Errorf("got prefix %q, want %q", authenticator.prefix, "abcdefgh")
	}
	if authenticator.keyHash != "secret" {
		t.Errorf("got key hash %q, want %q", authenticator.keyHash, "secret")
	}
}

func TestAuthHandler_AuthenticationSucceeded(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer abcdefgh.secret")
	rec := httptest.NewRecorder()
	apiKeyID := pgtype.UUID{
		Bytes: [16]byte{7, 8, 9},
		Valid: true,
	}
	authenticator := &testAPIKeyAuthenticator{
		updated: make(chan struct{}),
		apiKey: &sqlc.ApiKey{
			ID: apiKeyID,
			OrgID: pgtype.UUID{
				Bytes: [16]byte{1, 2, 3},
				Valid: true,
			},
			UserID: pgtype.UUID{
				Bytes: [16]byte{4, 5, 6},
				Valid: true,
			},
		},
	}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authenticator.authCalls != 1 {
		t.Fatalf("expected authenticator to be called once, got %d calls", authenticator.authCalls)
	}
	if authenticator.prefix != "abcdefgh" {
		t.Errorf("got prefix %q, want %q", authenticator.prefix, "abcdefgh")
	}
	if authenticator.keyHash != "secret" {
		t.Errorf("got key hash %q, want %q", authenticator.keyHash, "secret")
	}
	select {
	case <-authenticator.updated:
	case <-time.After(time.Second):
		t.Fatal("expected verification timestamp to be updated")
	}
	if authenticator.updateCalls != 1 {
		t.Fatalf("expected verification timestamp to be updated once, got %d calls", authenticator.updateCalls)
	}
	if authenticator.updatedID != apiKeyID {
		t.Errorf("got updated id %v, want %v", authenticator.updatedID, apiKeyID)
	}

}
