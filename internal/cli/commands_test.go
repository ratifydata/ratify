package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/spf13/cobra"
)

// Unconfigured operations fail immediately, so validation tests also prove
// that invalid input never reaches the inspector.
type stubInspector struct {
	t    *testing.T
	list func(context.Context) ([]schema.StoredConnection, error)
	test func(context.Context, pgtype.UUID) error
	add  func(context.Context, schema.ConnectionParams) error
}

func (s *stubInspector) ListDatabaseConnections(ctx context.Context) ([]schema.StoredConnection, error) {
	s.t.Helper()
	if s.list == nil {
		s.t.Fatal("unexpected ListDatabaseConnections call")
	}
	return s.list(ctx)
}
func (s *stubInspector) TestConnection(ctx context.Context, id pgtype.UUID) error {
	s.t.Helper()
	if s.test == nil {
		s.t.Fatal("unexpected TestConnection call")
	}
	return s.test(ctx, id)
}
func (s *stubInspector) SchemaInspection(ctx context.Context, params schema.ConnectionParams) error {
	s.t.Helper()
	if s.add == nil {
		s.t.Fatal("unexpected SchemaInspection call")
	}
	return s.add(ctx, params)
}

const testOrgID = "11111111-1111-4111-8111-111111111111"
const testConnectionID = "22222222-2222-4222-8222-222222222222"

type cliContextKey struct{}

func executeCLI(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.WithValue(t.Context(), cliContextKey{}, "preserved"))
	return output.String(), err
}

func assertCLIContext(t *testing.T, ctx context.Context, wantOrg bool) {
	t.Helper()
	if got := ctx.Value(cliContextKey{}); got != "preserved" {
		t.Errorf("caller context value = %v, want preserved", got)
	}
	if wantOrg {
		id, ok := ctx.Value("OrgID").(pgtype.UUID)
		if !ok || !id.Valid || id.String() != testOrgID {
			t.Errorf("organization context = %v, want %s", ctx.Value("OrgID"), testOrgID)
		}
	}
}

func connectForTest(t *testing.T, c *ConnectCmd) *cobra.Command {
	t.Helper()
	previous := orgID
	t.Cleanup(func() { orgID = previous })
	return c.Connect()
}

func TestListConnections(t *testing.T) {
	backendErr := errors.New("list failed")
	var id pgtype.UUID
	if err := id.Scan(testConnectionID); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		rows []schema.StoredConnection
		err  error
		want string
	}{
		{name: "no connections", want: "No saved connections.\n"},
		{name: "saved connections", rows: []schema.StoredConnection{
			{ID: id, DisplayName: "analytics", Host: "db.example", Port: 5432, DatabaseName: "warehouse", Status: "ACTIVE"},
			{ID: id, DisplayName: "reporting", Host: "localhost", Port: 5433, DatabaseName: "reports", Status: "INACTIVE"},
		}, want: testConnectionID + "\tanalytics\tdb.example:5432/warehouse\tACTIVE\n" + testConnectionID + "\treporting\tlocalhost:5433/reports\tINACTIVE\n"},
		{name: "inspector error", err: backendErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			inspector := &stubInspector{t: t, list: func(ctx context.Context) ([]schema.StoredConnection, error) {
				calls++
				assertCLIContext(t, ctx, true)
				return tt.rows, tt.err
			}}
			c := &ConnectCmd{inspector: inspector}
			got, err := executeCLI(t, connectForTest(t, c), "--org-id", testOrgID, "list")
			if !errors.Is(err, tt.err) {
				t.Errorf("Execute() error = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
			if calls != 1 {
				t.Errorf("list calls = %d, want 1", calls)
			}
		})
	}
}

func TestTestConnectionCommand(t *testing.T) {
	backendErr := errors.New("connection unavailable")
	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: "Connection test passed.\n"},
		{name: "inspector error", err: backendErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			inspector := &stubInspector{t: t, test: func(ctx context.Context, id pgtype.UUID) error {
				calls++
				assertCLIContext(t, ctx, false)
				if !id.Valid || id.String() != testConnectionID {
					t.Errorf("connection ID = %v, want %s", id, testConnectionID)
				}
				return tt.err
			}}
			c := &ConnectCmd{inspector: inspector}
			got, err := executeCLI(t, connectForTest(t, c), "--org-id", testOrgID, "test", testConnectionID)
			if !errors.Is(err, tt.err) {
				t.Errorf("Execute() error = %v, want %v", err, tt.err)
			}
			if got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
			if calls != 1 {
				t.Errorf("test calls = %d, want 1", calls)
			}
		})
	}
}

func TestAddConnectionCommand(t *testing.T) {
	promptErr := errors.New("input interrupted")
	backendErr := errors.New("inspection failed")
	for _, tt := range []struct {
		name, port, sslMode     string
		promptErr, inspectorErr error
		wantErr                 error
		wantErrText             string
		wantCalls               int
	}{
		{name: "SSL disabled", port: "5432", sslMode: string(schema.DISABLED), wantCalls: 1},
		{name: "SSL enabled", port: "5433", sslMode: string(schema.VERIFY_FULL), wantCalls: 1},
		{name: "prompt error", promptErr: promptErr, wantErr: promptErr},
		{name: "invalid port", port: "not-a-port", wantErrText: "invalid syntax"},
		{name: "inspector error", port: "5432", inspectorErr: backendErr, wantErr: backendErr, wantCalls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls, prompts := 0, 0
			inspector := &stubInspector{t: t, add: func(ctx context.Context, params schema.ConnectionParams) error {
				calls++
				assertCLIContext(t, ctx, true)
				port := 5432
				if tt.port == "5433" {
					port = 5433
				}
				want := schema.ConnectionParams{Host: "db.example", Port: port, Username: "reader", Password: "secret", DatabaseName: "warehouse", SSlEnabled: tt.sslMode != string(schema.DISABLED), DriverName: "pgx"}
				if params != want {
					t.Errorf("connection parameters = %+v, want %+v", params, want)
				}
				return tt.inspectorErr
			}}
			c := &ConnectCmd{inspector: inspector, promptDetails: func() (*ConnectionDetails, error) {
				prompts++
				return &ConnectionDetails{Host: "db.example", Port: tt.port, Username: "reader", Password: "secret", DatabaseName: "warehouse", SSLMode: tt.sslMode}, tt.promptErr
			}}
			_, err := executeCLI(t, connectForTest(t, c), "--org-id", testOrgID, "add")
			if tt.wantErrText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
					t.Errorf("Execute() error = %v, want containing %q", err, tt.wantErrText)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if calls != tt.wantCalls {
				t.Errorf("inspection calls = %d, want %d", calls, tt.wantCalls)
			}
			if prompts != 1 {
				t.Errorf("prompt calls = %d, want 1", prompts)
			}
		})
	}
}

func TestConnectionCommandsRejectInvalidOrganization(t *testing.T) {
	for _, subcommand := range []string{"list", "add"} {
		for _, tt := range []struct {
			name string
			args []string
			want string
		}{
			{"missing", nil, `required flag(s) "org-id" not set`},
			{"empty", []string{"--org-id", ""}, "organization ID is required"},
			{"malformed", []string{"--org-id", "not-a-uuid"}, "invalid organization ID"},
		} {
			t.Run(subcommand+"/"+tt.name, func(t *testing.T) {
				c := &ConnectCmd{inspector: &stubInspector{t: t}, promptDetails: func() (*ConnectionDetails, error) {
					t.Fatal("invalid organization must be rejected before prompting")
					return nil, nil
				}}
				_, err := executeCLI(t, connectForTest(t, c), append(tt.args, subcommand)...)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("Execute() error = %v, want containing %q", err, tt.want)
				}
			})
		}
	}
}
