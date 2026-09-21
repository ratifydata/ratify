package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ratifydata/ratify/internal/auth"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/ratifydata/ratify/internal/testutil"
	"github.com/spf13/cobra"
)

const connectionTestKey = "0123456789abcdef0123456789abcdef"

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
			got, err := prompt(bufio.NewReader(strings.NewReader(tt.input)), "Hostname")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("prompt() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("prompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectCommandValidation(t *testing.T) {
	// Validation must fail before any database access; a nil inspector makes
	// accidental execution of a database operation fail the test.
	for _, tt := range []struct {
		name       string
		args       []string
		wantErr    string
		wantOutput string
	}{
		{"missing connection", []string{"test"}, "accepts 1 arg(s), received 0", ""},
		{"extra connection", []string{"test", "one", "two"}, "accepts 1 arg(s), received 2", ""},
		{"invalid connection", []string{"test", "not-a-uuid"}, "", "failure\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			connect := &ConnectCmd{}
			output, err := executeCLI(t, connect.Connect(), tt.args...)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Execute() error = %v, want nil", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Execute() error = %v, want containing %q", err, tt.wantErr)
			}
			if output != tt.wantOutput {
				t.Errorf("output = %q, want %q", output, tt.wantOutput)
			}
		})
	}
}

func TestConnectionCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Postgres containers")
	}
	containers, err := testutil.InitializePostgresContainer()
	if err != nil {
		t.Fatalf("initialize Postgres containers: %v", err)
	}
	t.Cleanup(func() {
		containers.Internal.Pool.Close()
		if err := containers.External.DB.Close(); err != nil {
			t.Errorf("close external database: %v", err)
		}
		testutil.TerminateContainer(containers.Internal.Container, containers.External.Container)
	})
	queries := sqlc.New(containers.Internal.Pool)
	// Read the mapped endpoint and credentials from ClientExternalTestContainer,
	// rather than using the internal metadata database as the external database.
	external, err := pgx.ParseConfig(containers.External.DSN)
	if err != nil {
		t.Fatalf("parse external DSN: %v", err)
	}
	newOrg := func(t *testing.T) sqlc.Organization {
		t.Helper()
		org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: t.Name(), Slug: uuid.NewString()})
		if err != nil {
			t.Fatalf("create organization: %v", err)
		}
		return org
	}
	seed := func(t *testing.T, org sqlc.Organization, password string) sqlc.DatabaseConnection {
		t.Helper()
		encrypted, err := auth.Encrypt([]byte(connectionTestKey), password)
		if err != nil {
			t.Fatal(err)
		}
		connection, err := queries.CreateDatabaseConnection(t.Context(), sqlc.CreateDatabaseConnectionParams{
			OrgID: org.ID, DisplayName: "External warehouse", Host: external.Host, Port: int32(external.Port),
			DatabaseName: external.Database, Username: external.User, PasswordEncrypted: encrypted.CipherText,
			Nonce: encrypted.Nonce, SslMode: "disable", SslEnabled: false, Status: "ACTIVE",
		})
		if err != nil {
			t.Fatalf("create database connection: %v", err)
		}
		return connection
	}

	t.Run("list", func(t *testing.T) {
		org := newOrg(t)
		saved := seed(t, org, external.Password)
		// A second organization's connection must never appear in this list.
		seed(t, newOrg(t), external.Password)
		ctx := context.WithValue(t.Context(), "OrgID", org.ID)
		for _, format := range []string{"table", "json"} {
			t.Run(format, func(t *testing.T) {
				c := NewConnectCmd(queries, connectionTestKey)
				output, err := executeConnectionCLI(ctx, c.Connect(), "list", "--output", format)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if format == "table" {
					want := fmt.Sprintf("%s\t%s\t%s:%d/%s\t%s\n", saved.ID.String(), saved.DisplayName, saved.Host, saved.Port, saved.DatabaseName, saved.Status)
					if output != want {
						t.Errorf("output = %q, want %q", output, want)
					}
				} else {
					var got []schema.StoredConnection
					if err := json.Unmarshal([]byte(output), &got); err != nil {
						t.Fatalf("decode output: %v", err)
					}
					want := schema.StoredConnection{ID: saved.ID, DisplayName: saved.DisplayName, Host: saved.Host, Port: saved.Port, DatabaseName: saved.DatabaseName, Username: saved.Username, SSLMode: saved.SslMode, SSLEnabled: saved.SslEnabled, Status: saved.Status}
					if len(got) != 1 {
						t.Fatalf("connections = %d, want 1", len(got))
					}
					if got[0] != want {
						t.Errorf("connection = %+v, want %+v", got[0], want)
					}
				}
			})
		}
	})
	t.Run("list empty", func(t *testing.T) {
		org := newOrg(t)
		ctx := context.WithValue(t.Context(), "OrgID", org.ID)
		for _, tt := range []struct{ format, want string }{{"table", "No saved connections.\n"}, {"json", "[]\n"}} {
			t.Run(tt.format, func(t *testing.T) {
				c := NewConnectCmd(queries, connectionTestKey)
				output, err := executeConnectionCLI(ctx, c.Connect(), "list", "--output", tt.format)
				if err != nil {
					t.Fatal(err)
				}
				if output != tt.want {
					t.Errorf("output = %q, want %q", output, tt.want)
				}
			})
		}
	})
	t.Run("list missing organization", func(t *testing.T) {
		c := NewConnectCmd(queries, connectionTestKey)
		_, err := executeCLI(t, c.Connect(), "list")
		if err == nil || !strings.Contains(err.Error(), "OrgID missing from context") {
			t.Errorf("error = %v, want missing organization", err)
		}
	})
	t.Run("test", func(t *testing.T) {
		for _, tt := range []struct {
			name, password string
			pass           bool
		}{{"success", external.Password, true}, {"wrong password", "wrong-password", false}} {
			t.Run(tt.name, func(t *testing.T) {
				org := newOrg(t)
				saved := seed(t, org, tt.password)
				c := NewConnectCmd(queries, connectionTestKey)
				output, err := executeConnectionCLI(context.WithValue(t.Context(), "OrgID", org.ID), c.Connect(), "test", saved.ID.String())
				if err != nil {
					t.Fatalf("test command error = %v, want nil", err)
				}
				want := "failure\n"
				if tt.pass {
					want = "success\n"
				}
				if output != want {
					t.Errorf("output = %q, want %q", output, want)
				}
				got, err := queries.GetDatabaseConnection(t.Context(), saved.ID)
				if err != nil {
					t.Fatal(err)
				}
				if !got.LastTestedAt.Valid || !got.LastTestPassed.Valid || got.LastTestPassed.Bool != tt.pass {
					t.Errorf("test result not persisted: timestamp=%v passed=%v", got.LastTestedAt, got.LastTestPassed)
				}
			})
		}
	})
	t.Run("test missing connection", func(t *testing.T) {
		c := NewConnectCmd(queries, connectionTestKey)
		output, err := executeCLI(t, c.Connect(), "test", uuid.NewString())
		if err != nil {
			t.Fatalf("test command error = %v, want nil", err)
		}
		if output != "failure\n" {
			t.Errorf("output = %q, want failure", output)
		}
	})
	t.Run("add", func(t *testing.T) {
		org := newOrg(t)
		runAddConnectionTerminal(t, containers.Internal.Pool.Config().ConnString(), org.ID, external)
		rows, err := queries.ListDatabaseConnectionsByOrg(t.Context(), org.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatalf("saved connections = %d, want 1", len(rows))
		}
		saved, err := queries.GetDatabaseConnection(t.Context(), rows[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.OrgID != org.ID || saved.Host != external.Host || saved.Port != int32(external.Port) || saved.DatabaseName != external.Database || saved.Username != external.User || saved.SslEnabled || saved.Status != "ACTIVE" {
			t.Errorf("saved connection does not match external container: %+v", rows[0])
		}
		password, err := auth.Decrypt(saved.Nonce, saved.PasswordEncrypted, []byte(connectionTestKey))
		if err != nil {
			t.Fatalf("decrypt saved password: %v", err)
		}
		if password != external.Password {
			t.Error("saved password does not match external container")
		}
		c := NewConnectCmd(queries, connectionTestKey)
		output, err := executeCLI(t, c.Connect(), "test", saved.ID.String())
		if err != nil {
			t.Fatalf("test newly added connection: %v", err)
		}
		if output != "success\n" {
			t.Errorf("newly added connection output = %q, want success", output)
		}
	})
}

// The subprocess owns stdin (fd 0), as required by term.ReadPassword. It uses
// the same metadata container and organization as its parent test.
func TestAddConnectionProcess(t *testing.T) {
	dsn := os.Getenv("RATIFY_CONNECTION_TEST_DSN")
	if dsn == "" {
		return
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var org pgtype.UUID
	if err := org.Scan(os.Getenv("RATIFY_CONNECTION_TEST_ORG")); err != nil {
		t.Fatal(err)
	}
	c := NewConnectCmd(sqlc.New(pool), connectionTestKey)
	if _, err := executeConnectionCLI(context.WithValue(ctx, "OrgID", org), c.Connect(), "add"); err != nil {
		t.Fatalf("add connection: %v", err)
	}
}

func runAddConnectionTerminal(t *testing.T, dsn string, org pgtype.UUID, external *pgx.ConnConfig) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal("interactive Add integration test requires python3 for its pseudo-terminal")
	}
	answers, err := json.Marshal([]string{external.Host, strconv.Itoa(int(external.Port)), external.Database, external.User, "", external.Password, external.Password})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-c", connectionTerminalScript, os.Args[0], string(answers))
	cmd.Env = append(os.Environ(), "RATIFY_CONNECTION_TEST_DSN="+dsn, "RATIFY_CONNECTION_TEST_ORG="+org.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("interactive Add failed: %v\n%s", err, output)
	}
}

// Python's standard-library PTY support works on macOS and Linux without
// adding a Go dependency or changing the production password reader. Each
// answer waits for its prompt, so bufio cannot consume later password input.
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
