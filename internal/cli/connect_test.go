package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ratifydata/ratify/internal/schema"
	"github.com/spf13/cobra"
)

func executeCLI(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	return executeConnectionCLI(t.Context(), cmd, args...)
}
func executeConnectionCLI(ctx context.Context, cmd *cobra.Command, args ...string) (string, error) {
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(ctx)
	return output.String(), err
}

func TestPrompt(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
		wantErr           error
	}{
		{"first line", "localhost\n5432\n", "localhost", nil},
		{"whitespace", " \tlocalhost \t\n", "localhost", nil},
		{"windows newline", "localhost\r\n", "localhost", nil},
		{"blank", "\n", "", nil},
		{"empty input", "", "", io.EOF},
		{"unterminated input", "localhost", "", io.EOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prompt(bufio.NewReader(strings.NewReader(tt.input)), io.Discard, "Hostname")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("prompt() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("prompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectionHTTPCommands(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	for _, tt := range []struct {
		name                     string
		args                     []string
		method, path, body, want string
		status                   int
		wantErr                  string
	}{
		{name: "list table", args: []string{"connect", "list"}, method: "GET", path: "/api/cli/v1/connections", body: `[{"id":"` + id + `","display_name":"warehouse","host":"db","port":5432,"database_name":"analytics","status":"ACTIVE"}]`, want: "warehouse\tdb:5432/analytics\tACTIVE"},
		{name: "list json", args: []string{"connect", "list", "--output", "json"}, method: "GET", path: "/api/cli/v1/connections", body: `[]`, want: "[]\n"},
		{name: "list empty", args: []string{"connect", "list"}, method: "GET", path: "/api/cli/v1/connections", body: `[]`, want: "No saved connections."},
		{name: "test", args: []string{"connect", "test", id}, method: "POST", path: "/api/cli/v1/connections/" + id + "/test", body: `{"status":"ok"}`, want: "success"},
		{name: "test json", args: []string{"connect", "test", id, "--output", "json"}, method: "POST", path: "/api/cli/v1/connections/" + id + "/test", body: `{"status":"ok"}`, want: "\"status\": \"ok\""},
		{name: "test invalid json", args: []string{"connect", "test", id}, method: "POST", path: "/api/cli/v1/connections/" + id + "/test", body: `not JSON`, wantErr: "decode API response"},
		{name: "test unsuccessful status", args: []string{"connect", "test", id}, method: "POST", path: "/api/cli/v1/connections/" + id + "/test", body: `{"status":"err","message":"failed"}`, wantErr: "unexpected connection response"},
		{name: "add invalid output", args: []string{"connect", "add", "--output", "bad"}, wantErr: "output must be text or json"},
		{name: "test failure", args: []string{"connect", "test", id}, method: "POST", path: "/api/cli/v1/connections/" + id + "/test", body: `{"message":"connection failed"}`, status: 500, wantErr: "connection failed"},
		{name: "invalid id", args: []string{"connect", "test", "bad"}, wantErr: "invalid connection UUID"},
		{name: "invalid output", args: []string{"connect", "list", "--output", "bad"}, wantErr: "output must be"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method || r.URL.Path != tt.path || r.Header.Get("Authorization") != "Bearer abcdefgh.secret" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			writeCLIConfig(t, "API_KEY: abcdefgh.secret\n")
			t.Setenv("RATIFY_API_URL", server.URL)
			output, err := executeCLI(t, newRootCmd(), tt.args...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output, tt.want) {
				t.Fatalf("output = %q", output)
			}
		})
	}
}

func TestAddConnectionHTTP(t *testing.T) {
	received := make(chan schema.ConnectionParams, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/cli/v1/connections" || r.Header.Get("Authorization") != "Bearer abcdefgh.secret" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var params schema.ConnectionParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		received <- params
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer server.Close()
	writeCLIConfig(t, "API_KEY: abcdefgh.secret\n")
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	answers, _ := json.Marshal([]string{"db.example.com", "5433", "analytics", "reader", "require", "password", "password"})
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-c", connectionTerminalScript, os.Args[0], string(answers))
	cmd.Env = append(os.Environ(), "RATIFY_ADD_HTTP_TEST=1", "RATIFY_API_URL="+server.URL)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("interactive add: %v\n%s", err, output)
	}
	select {
	case params := <-received:
		if params.Host != "db.example.com" || params.Port != 5433 || params.Password != "password" || params.DatabaseName != "analytics" || params.Username != "reader" || params.SSLMode != "require" || !params.SSlEnabled || params.DriverName != "pgx" {
			t.Fatalf("wrong request: host=%s port=%d ssl=%s", params.Host, params.Port, params.SSLMode)
		}
	default:
		t.Fatal("no HTTP request")
	}
}
func TestAddConnectionProcess(t *testing.T) {
	if os.Getenv("RATIFY_ADD_HTTP_TEST") == "" {
		return
	}
	cmd := newRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(os.Stderr)
	cmd.SetArgs([]string{"connect", "add", "--output", "json"})
	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	var result connectionResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not JSON: %q: %v", output.String(), err)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q", result.Status)
	}
}

const connectionTerminalScript = `
import json, os, pty, select, subprocess, sys, termios, time
master, slave = pty.openpty()
p = subprocess.Popen([sys.argv[1], '-test.run=^TestAddConnectionProcess$'], stdin=slave, stdout=slave, stderr=slave)
answers = json.loads(sys.argv[2])
prompts = [b'Hostname: ', b'Port [5432]: ', b'Database: ', b'Username: ', b'SSL mode [disable]: ', b'Password: ', b'Confirm Password: ']
buffer = b''
step = 0
deadline = time.monotonic() + 35
try:
    while time.monotonic() < deadline:
        ready, _, _ = select.select([master], [], [], 0.1)
        if ready:
            chunk = os.read(master, 4096)
            sys.stdout.buffer.write(chunk)
            sys.stdout.buffer.flush()
            buffer += chunk
        if step < len(prompts) and prompts[step] in buffer:
            # Wait until ReadPassword disables echo before sending a password.
            if step < 5 or not (termios.tcgetattr(slave)[3] & termios.ECHO):
                os.write(master, (answers[step] + '\n').encode())
                buffer = b''
                step += 1
        if p.poll() is not None:
            # Drain final test failures before reporting the child's exit code.
            while select.select([master], [], [], 0)[0]:
                sys.stdout.buffer.write(os.read(master, 4096))
            if step != len(prompts):
                raise RuntimeError('child exited before completing all prompts')
            sys.exit(p.returncode)
    raise TimeoutError('timed out waiting for interactive Add')
finally:
    if p.poll() is None:
        p.kill()
    p.wait()
    os.close(master)
    os.close(slave)
`
