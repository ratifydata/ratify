package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type fakeAPIKeyAuthenticator struct {
	apiKey *sqlc.ApiKey
	err    error

	calls   int
	prefix  string
	keyHash string
}

func (f *fakeAPIKeyAuthenticator) ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error) {
	f.calls++
	f.prefix = prefix
	f.keyHash = keyHash

	if f.err != nil {
		return nil, f.err
	}

	return f.apiKey, nil
}

func TestAuthHandler_AuthorizationHeaderMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	authenticator := &fakeAPIKeyAuthenticator{}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := apiKeyAuthHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.calls != 0 {
		t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.calls)
	}
}

func TestAuthHandler_InvalidAuthorizationScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic abcdefgh.secret")
	rec := httptest.NewRecorder()
	authenticator := &fakeAPIKeyAuthenticator{}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := apiKeyAuthHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.calls != 0 {
		t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.calls)
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
			authenticator := &fakeAPIKeyAuthenticator{}
			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := apiKeyAuthHandler(authenticator)(next)
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if nextCalled {
				t.Fatal("expected next handler not to be called")
			}
			if authenticator.calls != 0 {
				t.Fatalf("expected authenticator not to be called, got %d calls", authenticator.calls)
			}
		})
	}
}

func TestAuthHandler_AuthenticationFailed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer abcdefgh.secret")
	rec := httptest.NewRecorder()
	authenticator := &fakeAPIKeyAuthenticator{err: errors.New("invalid api key")}
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := apiKeyAuthHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if authenticator.calls != 1 {
		t.Fatalf("expected authenticator to be called once, got %d calls", authenticator.calls)
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
	authenticator := &fakeAPIKeyAuthenticator{
		apiKey: &sqlc.ApiKey{
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

	handler := apiKeyAuthHandler(authenticator)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authenticator.calls != 1 {
		t.Fatalf("expected authenticator to be called once, got %d calls", authenticator.calls)
	}
	if authenticator.prefix != "abcdefgh" {
		t.Errorf("got prefix %q, want %q", authenticator.prefix, "abcdefgh")
	}
	if authenticator.keyHash != "secret" {
		t.Errorf("got key hash %q, want %q", authenticator.keyHash, "secret")
	}
	if rec.Header().Get("x-org-id") == "" {
		t.Fatal("expected x-org-id header to be set")
	}
	if rec.Header().Get("x-user-id") == "" {
		t.Fatal("expected x-user-id header to be set")
	}

}
