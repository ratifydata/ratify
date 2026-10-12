package cli

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestBridgeConfigure(t *testing.T) {
	for _, tt := range []struct{ name, config, env, flag, want, wantErr string }{
		{name: "default", config: "API_KEY: key", want: "http://localhost:8080"},
		{name: "saved", config: "API_KEY: key\nAPI_URL: https://saved.example/proxy/", want: "https://saved.example/proxy"},
		{name: "env", config: "API_KEY: key\nAPI_URL: https://saved.example", env: "https://env.example", want: "https://env.example"},
		{name: "flag", config: "API_KEY: key", env: "https://env.example", flag: "https://flag.example", want: "https://flag.example"},
		{name: "malformed", config: "API_KEY: [secret", wantErr: "invalid CLI config"},
		{name: "missing key", config: "{}", wantErr: "missing or invalid API_KEY"},
		{name: "invalid url", config: "API_KEY: key", flag: "ftp://host", wantErr: "invalid API URL"},
		{name: "url credentials", config: "API_KEY: key", flag: "http://user:secret@host", wantErr: "invalid API URL"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writeCLIConfig(t, tt.config)
			t.Setenv("RATIFY_API_URL", tt.env)
			cmd := &cobra.Command{}
			cmd.Flags().String("api-url", tt.flag, "")
			cmd.SetContext(t.Context())
			b := NewBridge()
			err := b.Configure(cmd)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.baseURL != tt.want {
				t.Fatalf("URL = %q", b.baseURL)
			}
			if cmd.Context().Value("OrgID") != nil {
				t.Fatal("organization assigned locally")
			}
		})
	}
}

func TestBridgeHTTPFailures(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		body, want string
	}{
		{"unauthorized", 401, "Unauthorized", "API authentication failed"},
		{"forbidden", 403, "Forbidden", "403: Forbidden"},
		{"json conflict", 409, `{"message":"team already exists"}`, "409: team already exists"},
		{"text failure", 500, "database unavailable", "500: database unavailable"},
		{"redaction", 400, "abcdefgh.secret", "[redacted]"},
		{"invalid response", 200, "not JSON", "decode API response"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer s.Close()
			b := NewBridge()
			b.baseURL = s.URL
			b.apiKey = "abcdefgh.secret"
			var response any
			err := b.Do(t.Context(), "GET", "/teams", nil, &response)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBridgeCancellationTimeoutAndRedirect(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	b := NewBridge()
	b.baseURL = s.URL
	b.apiKey = "abcdefgh.secret"
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := b.Do(ctx, "GET", "/teams", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	b.client.Timeout = 20 * time.Millisecond
	if err := b.Do(t.Context(), "GET", "/teams", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("redirect followed") }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	b.baseURL = redirect.URL
	b.client.Timeout = time.Second
	if err := b.Do(t.Context(), "POST", "/teams", nil, nil); err == nil || !strings.Contains(err.Error(), "307") {
		t.Fatalf("error = %v", err)
	}
}

func TestRootRequiresConfigForHTTPCommands(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := executeCLI(t, newRootCmd(), "connect", "list")
	if err == nil || !strings.Contains(err.Error(), "read CLI config") {
		t.Fatalf("error = %v", err)
	}
}
