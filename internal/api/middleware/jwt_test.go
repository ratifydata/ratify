package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/config"
)

const testJWTSecret = "test-jwt-secret"

func testJWTMiddleware(next http.Handler) http.Handler {
	return JwtAuthMiddleware(&config.Config{JWTSecret: testJWTSecret})(next)
}

func TestJwtAuthMiddleware_AuthorizationHeaderMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := testJWTMiddleware(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
}

func TestJwtAuthMiddleware_InvalidAuthorizationScheme(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Basic invalid-token")
	rec := httptest.NewRecorder()
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := testJWTMiddleware(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
}

func TestJwtAuthMiddleware_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := testJWTMiddleware(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
}

func TestJwtAuthMiddleware_ValidToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	token, err := auth.GenerateJWT(userID, orgID, testJWTSecret)
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if got := r.Context().Value(OrgId); got != orgID {
			t.Errorf("got org ID %v in context, want %s", got, orgID)
		}
		if got := r.Context().Value(UserId); got != userID {
			t.Errorf("got user ID %v in context, want %s", got, userID)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := testJWTMiddleware(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rec.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}
