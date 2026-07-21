package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ratifydata/ratify/internal/auth"
)

type fakeLoginService struct {
	token string
	err   error

	calls  int
	params auth.LoginParams
}

func (f *fakeLoginService) AuthenticateUser(ctx context.Context, params auth.LoginParams) (string, error) {
	f.calls++
	f.params = params

	if f.err != nil {
		return "", f.err
	}

	return f.token, nil
}

func TestLoginHandler_SuccessSetsHTTPOnlyJWTCookie(t *testing.T) {
	service := &fakeLoginService{token: "signed.jwt.token"}
	body := bytes.NewBufferString(`{"username":"user@example.com","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	rec := httptest.NewRecorder()

	loginHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}
	if service.calls != 1 {
		t.Fatalf("expected login service to be called once, got %d calls", service.calls)
	}
	if service.params.Username != "user@example.com" {
		t.Errorf("got username %q, want %q", service.params.Username, "user@example.com")
	}
	if service.params.Password != "secret" {
		t.Errorf("got password %q, want %q", service.params.Password, "secret")
	}

	cookie := findCookie(t, rec.Result(), jwtCookieName)
	if cookie.Value != "signed.jwt.token" {
		t.Errorf("got cookie value %q, want %q", cookie.Value, "signed.jwt.token")
	}
	if !cookie.HttpOnly {
		t.Fatal("expected jwt cookie to be HTTP-only")
	}
	if cookie.Path != "/" {
		t.Errorf("got cookie path %q, want %q", cookie.Path, "/")
	}
	if cookie.MaxAge != 24*60*60 {
		t.Errorf("got cookie max age %d, want %d", cookie.MaxAge, 24*60*60)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("got SameSite %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message != "login successful" {
		t.Errorf("got response message %q, want %q", resp.Message, "login successful")
	}
}

func TestLoginHandler_InvalidJSON(t *testing.T) {
	service := &fakeLoginService{token: "signed.jwt.token"}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	loginHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if service.calls != 0 {
		t.Fatalf("expected login service not to be called, got %d calls", service.calls)
	}
}

func TestLoginHandler_LoginFailed(t *testing.T) {
	service := &fakeLoginService{err: errors.New("invalid credentials")}
	body := bytes.NewBufferString(`{"username":"user@example.com","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	rec := httptest.NewRecorder()

	loginHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if service.calls != 1 {
		t.Fatalf("expected login service to be called once, got %d calls", service.calls)
	}
	if cookie := findOptionalCookie(rec.Result(), jwtCookieName); cookie != nil {
		t.Fatalf("expected no jwt cookie, got %+v", cookie)
	}
}

func TestLoginHandler_SecureCookieWhenRequestIsTLS(t *testing.T) {
	service := &fakeLoginService{token: "signed.jwt.token"}
	req := httptest.NewRequest(http.MethodPost, "https://example.com/auth/login", bytes.NewBufferString(`{"username":"user@example.com","password":"secret"}`))
	rec := httptest.NewRecorder()

	loginHandler(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	cookie := findCookie(t, rec.Result(), jwtCookieName)
	if !cookie.Secure {
		t.Fatal("expected jwt cookie to be secure for TLS requests")
	}
}

func TestNewRouter_RegistersAuthLoginRoute(t *testing.T) {
	router := NewRouter(nil)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"username":"user@example.com","password":"secret"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("expected POST /auth/login to be registered")
	}
	if rec.Code == http.StatusMethodNotAllowed {
		t.Fatal("expected POST /auth/login to allow POST requests")
	}
}

func findCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	cookie := findOptionalCookie(resp, name)
	if cookie == nil {
		t.Fatalf("expected %q cookie to be set", name)
	}

	return cookie
}

func findOptionalCookie(resp *http.Response, name string) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}
