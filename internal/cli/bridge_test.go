package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestBridgeConfigure(t *testing.T) {
	for _, tt := range []struct{ name, config, env, flag, want, wantErr string }{
		{name: "default", config: "API_KEY: key", want: "http://localhost:8080"},
		{name: "saved", config: "API_KEY: key\nAPI_URL: https://saved.example/proxy/", want: "https://saved.example/proxy"},
		{name: "environment ignored", config: "API_KEY: key\nAPI_URL: https://saved.example", env: "https://env.example", want: "https://saved.example"},
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
	if err := b.Do(t.Context(), "POST", "/teams", nil, nil); err != nil {
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

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *trackedBody) Close() error { b.closed = true; return b.closeErr }

func TestBridgeRequestAndResponse(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(`{"name":"DATA"}`)}
			type contextKey struct{}
			ctx := context.WithValue(t.Context(), contextKey{}, "retained")
			var input any
			if method == http.MethodPost {
				input = map[string]string{"name": "Data & Platform"}
			}
			b := NewBridge()
			b.baseURL = "https://example.com/proxy"
			b.apiKey = "abcdefgh.secret"
			b.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://example.com/proxy/api/cli/v1/teams" || r.Method != method {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if r.Context().Value(contextKey{}) != "retained" {
					t.Error("request lost context")
				}
				if r.Header.Get("Authorization") != "Bearer abcdefgh.secret" || r.Header.Get("Accept") != "application/json" {
					t.Error("missing request headers")
				}
				if input == nil {
					if r.Body != nil || r.Header.Get("Content-Type") != "" {
						t.Error("unexpected GET body or content type")
					}
				} else {
					var got map[string]string
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Error(err)
					}
					if got["name"] != "Data & Platform" || r.Header.Get("Content-Type") != "application/json" {
						t.Error("incorrect JSON request")
					}
				}
				return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
			})
			var result struct {
				Name string `json:"name"`
			}
			if err := b.Do(ctx, method, "/teams", input, &result); err != nil {
				t.Fatal(err)
			}
			if result.Name != "DATA" || !body.closed {
				t.Fatalf("result=%+v closed=%v", result, body.closed)
			}
		})
	}
}

func TestBridgeResponseHandling(t *testing.T) {
	// The bridge now leaves status interpretation to callers and only decodes JSON.
	for _, tt := range []struct {
		name                string
		status              int
		body                string
		discard, closeError bool
		wantErr             string
	}{
		{name: "success", status: 200, body: `{"status":"ok"}`},
		{name: "JSON HTTP error passed to caller", status: 500, body: `{"status":"err"}`},
		{name: "plain HTTP error is a decode error", status: 401, body: "Unauthorized", wantErr: "decode API response"},
		{name: "malformed JSON", status: 200, body: "{", wantErr: "decode API response"},
		{name: "empty decoded body", status: 204, wantErr: "decode API response"},
		{name: "discard empty body", status: 204, discard: true},
		{name: "discard HTTP error body", status: 500, body: "failure", discard: true},
		{name: "close failure is logged", status: 200, body: `{"status":"ok"}`, closeError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := &trackedBody{Reader: strings.NewReader(tt.body)}
			if tt.closeError {
				body.closeErr = errors.New("close failed")
			}
			b := NewBridge()
			b.baseURL = "https://example.com"
			b.client.Transport = testTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Body: body, Header: make(http.Header)}, nil
			})
			var got map[string]string
			var output any = &got
			if tt.discard {
				output = nil
			}
			err := b.Do(t.Context(), "GET", "/teams", nil, output)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if !body.closed {
				t.Error("response body not closed")
			}
			if err == nil && !tt.discard {
				var want map[string]string
				_ = json.Unmarshal([]byte(tt.body), &want)
				if got["status"] != want["status"] {
					t.Errorf("decoded response=%v", got)
				}
			}
		})
	}
}

func TestBridgeConfigureValidation(t *testing.T) {
	for _, tt := range []struct{ name, config, flag, want string }{
		{"blank key", "API_KEY: '   '", "", "missing or invalid API_KEY"},
		{"header injection", "API_KEY: \"key\\nInjected: value\"", "", "missing or invalid API_KEY"},
		{"invalid key type", "API_KEY: [one, two]", "", "invalid CLI config"},
		{"URL no host", "API_KEY: key", "https:///path", "invalid API URL"},
		{"URL query", "API_KEY: key", "https://example.com?token=x", "invalid API URL"},
		{"URL fragment", "API_KEY: key", "https://example.com#fragment", "invalid API URL"},
		{"URL malformed", "API_KEY: key", "http://%zz", "invalid API URL"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writeCLIConfig(t, tt.config)
			cmd := &cobra.Command{}
			cmd.Flags().String("api-url", tt.flag, "")
			err := NewBridge().Configure(cmd)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	t.Run("missing config", func(t *testing.T) {
		t.Chdir(t.TempDir())
		err := NewBridge().Configure(&cobra.Command{})
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("missing flag", func(t *testing.T) {
		writeCLIConfig(t, "API_KEY: key")
		if err := NewBridge().Configure(&cobra.Command{}); err == nil {
			t.Fatal("expected missing flag error")
		}
	})
	t.Run("trim credentials", func(t *testing.T) {
		writeCLIConfig(t, "API_KEY: '  abcdefgh.secret  '\nAPI_URL: ' https://example.com/base/// '")
		cmd := &cobra.Command{}
		cmd.Flags().String("api-url", "", "")
		b := NewBridge()
		if err := b.Configure(cmd); err != nil {
			t.Fatal(err)
		}
		if b.apiKey != "abcdefgh.secret" || b.baseURL != "https://example.com/base" {
			t.Fatal("configuration not trimmed")
		}
	})
}
