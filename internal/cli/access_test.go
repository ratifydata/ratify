package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ratifydata/ratify/internal/access"
	"github.com/ratifydata/ratify/internal/api"

	"github.com/google/uuid"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/config"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/testutil"
)

func TestAddTeamValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"missing name", nil, "team name is required"},
		{"blank name", []string{"--name", "  "}, "team name is required"},
		{"missing email", []string{"--name", "Data"}, "email address is required"},
		{"extra arg", []string{"unexpected"}, "unknown command"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &AccessCmd{}
			_, err := executeCLI(t, c.TeamCmd(), append([]string{"add"}, tt.args...)...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestAddTeamCommand(t *testing.T) {
	containers, err := testutil.InitializePostgresContainer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		containers.Internal.Pool.Close()
		_ = containers.External.DB.Close()
		testutil.TerminateContainer(containers.Internal.Container, containers.External.Container)
	})
	queries := sqlc.New(containers.Internal.Pool)
	org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "CLI teams", Slug: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	user, err := queries.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: "cli@example.com", DisplayName: "CLI user", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := auth.Hash("cli-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = queries.CreateAPIKey(t.Context(), sqlc.CreateAPIKeyParams{OrgID: org.ID, UserID: user.ID, Name: "CLI test", KeyPrefix: "clitest1", KeyHash: hash, Scope: "write", IsActive: true})
	if err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter(containers.Internal.Pool, &config.Config{EncryptionKey: "0123456789abcdef0123456789abcdef"})
	var responseStatus atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, r)
		responseStatus.Store(int32(response.Code))
		for name, values := range response.Header() {
			w.Header()[name] = values
		}
		w.WriteHeader(response.Code)
		_, _ = w.Write(response.Body.Bytes())
	}))
	t.Cleanup(server.Close)
	writeCLIConfig(t, "API_KEY: clitest1.cli-test-secret\nAPI_URL: "+server.URL+"\n")
	for _, name := range []string{"Data Team", "Context Team"} {
		args := []string{"add", "--email", "team@example.com", "--description", "Owns pipelines"}
		args = append(args, "--name", name)
		service := access.NewTeam(queries)
		output, err := executeCLI(t, newRootCmd(), append([]string{"team"}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		teams, err := service.ListTeams(context.WithValue(t.Context(), "OrgID", org.ID))
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, team := range teams {
			if team.Name == strings.ToUpper(strings.ReplaceAll(name, " ", "")) {
				found = true
				if team.EmailAddress != "team@example.com" || team.Description != "Owns pipelines" || !strings.Contains(output, team.ID.String()) {
					t.Fatalf("unexpected team or output: %+v, %q", team, output)
				}
			}
		}
		if !found {
			t.Fatalf("created team not persisted: %q", name)
		}
		// The bridge decodes JSON regardless of HTTP status. Verify server rejection
		// and persistence rather than expecting the former bridge error text.
		_, _ = executeCLI(t, newRootCmd(), append([]string{"team"}, args...)...)
		if got := responseStatus.Load(); got != http.StatusConflict {
			t.Fatalf("duplicate HTTP status = %d, want %d", got, http.StatusConflict)
		}
		after, err := queries.ListTeamsByOrg(t.Context(), org.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(teams) {
			t.Fatalf("duplicate changed team count: got %d, want %d", len(after), len(teams))
		}
	}
	for _, key := range []string{"clitest1.wrong-secret", "unknown1.secret"} {
		if err := os.WriteFile(".ratify/config.yml", []byte("API_KEY: "+key+"\nAPI_URL: "+server.URL+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := executeCLI(t, newRootCmd(), "team", "add", "--name", "Rejected", "--email", "team@example.com")
		if got := responseStatus.Load(); got != http.StatusUnauthorized {
			t.Fatalf("invalid key HTTP status = %d, want %d", got, http.StatusUnauthorized)
		}
		// The middleware's plain-text 401 body is rejected by the JSON decoder.
		if err == nil || !strings.Contains(err.Error(), "decode API response") {
			t.Fatalf("invalid key error = %v", err)
		}
	}
	teams, err := queries.ListTeamsByOrg(t.Context(), org.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("unexpected persisted teams: count=%d err=%v", len(teams), err)
	}

}

func TestTeamListOutput(t *testing.T) {
	for _, tt := range []struct{ name, body, format, want string }{
		{"table", `[{"name":"DATA","description":"Pipelines","email_address":"data@example.com"}]`, "table", "DATA\tPipelines\tdata@example.com\n\n"},
		{"json", `[{"name":"DATA","description":"Pipelines","email_address":"data@example.com"}]`, "json", "DATA"},
		{"empty table", `[]`, "table", ""},
		{"empty json", `[]`, "json", "[]\n"},
		{"null json", `null`, "json", "[]\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBridge()
			b.baseURL = "https://example.com"
			b.apiKey = "abcdefgh.secret"
			calls := 0
			b.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/api/cli/v1/teams" {
					t.Errorf("request=%s %s", r.Method, r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tt.body)), Header: make(http.Header)}, nil
			})
			c := NewAccessCmd(b)
			output, err := executeCLI(t, c.TeamCmd(), "list", "--output", tt.format)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("calls=%d", calls)
			}
			if tt.name == "json" {
				var got []access.OrgTeam
				if err := json.Unmarshal([]byte(output), &got); err != nil {
					t.Fatal(err)
				}
				if len(got) != 1 || got[0].Name != "DATA" || got[0].Description != "Pipelines" || got[0].EmailAddress != "data@example.com" {
					t.Fatalf("teams=%+v", got)
				}
			} else if output != tt.want {
				t.Fatalf("output=%q, want %q", output, tt.want)
			}
		})
	}
}

func TestTeamAddRequest(t *testing.T) {
	b := NewBridge()
	b.baseURL = "https://example.com"
	b.apiKey = "abcdefgh.secret"
	b.client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" || r.URL.Path != "/api/cli/v1/teams" {
			t.Errorf("request=%s %s", r.Method, r.URL)
		}
		var params access.TeamParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Error(err)
		}
		if params.Name != "Data Platform" || params.EmailAddress != "data@example.com" || params.Description != "Pipelines" {
			t.Errorf("params=%+v", params)
		}
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"status":"ok","body":{"id":"11111111-1111-1111-1111-111111111111","name":"DATAPLATFORM"}}`)), Header: make(http.Header)}, nil
	})
	c := NewAccessCmd(b)
	output, err := executeCLI(t, c.TeamCmd(), "add", "--name", " Data Platform ", "--email", " data@example.com ", "--description", "Pipelines")
	if err != nil {
		t.Fatal(err)
	}
	if output != "Created team DATAPLATFORM (11111111-1111-1111-1111-111111111111)\n" {
		t.Fatalf("output=%q", output)
	}
}

func TestTeamListRejectsArguments(t *testing.T) {
	c := NewAccessCmd(nil)
	if _, err := executeCLI(t, c.ListTeam(), "unexpected"); err == nil {
		t.Fatal("accepted positional argument")
	}
}
