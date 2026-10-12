package cli

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	if testing.Short() {
		t.Skip("requires Postgres containers")
	}
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
	server := httptest.NewServer(api.NewRouter(containers.Internal.Pool, &config.Config{EncryptionKey: "0123456789abcdef0123456789abcdef"}))
	t.Cleanup(server.Close)
	t.Setenv("RATIFY_API_URL", server.URL)
	writeCLIConfig(t, "API_KEY: clitest1.cli-test-secret\n")
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
		_, err = executeCLI(t, newRootCmd(), append([]string{"team"}, args...)...)
		if err == nil || !strings.Contains(err.Error(), "team already exists") {
			t.Fatalf("duplicate error = %v", err)
		}
	}
	// Exercise connection listing and the chi path parameter through the real router.
	external, err := pgx.ParseConfig(containers.External.DSN)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := auth.Encrypt([]byte("0123456789abcdef0123456789abcdef"), external.Password)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := queries.CreateDatabaseConnection(t.Context(), sqlc.CreateDatabaseConnectionParams{
		OrgID: org.ID, DisplayName: "HTTP warehouse", Host: external.Host, Port: int32(external.Port), DatabaseName: external.Database,
		Username: external.User, PasswordEncrypted: encrypted.CipherText, Nonce: encrypted.Nonce, SslMode: "disable", Status: "ACTIVE",
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := executeCLI(t, newRootCmd(), "connect", "list", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	var connections []struct {
		ID pgtype.UUID `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &connections); err != nil {
		t.Fatal(err)
	}
	if len(connections) != 1 || connections[0].ID != saved.ID {
		t.Fatalf("connections = %s", output)
	}
	output, err = executeCLI(t, newRootCmd(), "connect", "test", saved.ID.String())
	if err != nil || output != "success\n" {
		t.Fatalf("test: output=%q err=%v", output, err)
	}
	for _, key := range []string{"clitest1.wrong-secret", "unknown1.secret"} {
		if err := os.WriteFile(".ratify/config.yml", []byte("API_KEY: "+key+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := executeCLI(t, newRootCmd(), "team", "add", "--name", "Rejected", "--email", "team@example.com")
		if err == nil || !strings.Contains(err.Error(), "401") {
			t.Fatalf("invalid key error = %v", err)
		}
	}
	teams, err := queries.ListTeamsByOrg(t.Context(), org.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("unexpected persisted teams: count=%d err=%v", len(teams), err)
	}

}
