package access

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatString(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"uppercase", "engineering", "ENGINEERING"},
		{"remove spaces", "  Data Platform  ", "DATAPLATFORM"},
		{"empty", "", ""},
		{"preserve punctuation", "api-team", "API-TEAM"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatString(tt.input); got != tt.want {
				t.Errorf("formatString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateTeam(t *testing.T) {
	test := []struct {
		name     string
		input    string
		hasError bool
	}{
		{
			name:     "create team",
			input:    "Data Platform",
			hasError: false,
		}, {
			name:     "Create Duplicate",
			input:    "Data Platform",
			hasError: true,
		},
	}
	queries := teamTestQueries(t)
	org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{
		Name: "Create Team Organization",
		Slug: "create-team-org",
	})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	service := NewTeam(queries)

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			_, err = service.CreateTeam(ctx, TeamParams{
				Name:        " Data Platform ",
				Description: "Owns data pipelines",
			})

			if tt.hasError != (err != nil) {
				require.Equal(t, tt.hasError, err != nil)
			}
		})
	}
}

func TestCreateTeamMissingOrganization(t *testing.T) {
	team, err := NewTeam(nil).CreateTeam(t.Context(), TeamParams{Name: "Engineering"})
	require.EqualError(t, err, "OrgID missing from context")
	assert.Nil(t, team)
}

func TestGetTeam(t *testing.T) {
	queries := teamTestQueries(t)
	org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{
		Name: "Get Team Organization",
		Slug: "get-team-org",
	})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	stored, err := queries.CreateTeam(ctx, sqlc.CreateTeamParams{
		OrgID:       org.ID,
		Name:        "ENGINEERING",
		Description: pgtype.Text{String: "Builds the platform", Valid: true},
	})
	require.NoError(t, err)

	team, err := NewTeam(queries).GetTeam(ctx, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, team)
	assert.Equal(t, stored.ID, team.ID)
	assert.Equal(t, "ENGINEERING", team.Name)
	assert.Equal(t, "Builds the platform", team.Description)

	t.Run("not found", func(t *testing.T) {
		team, err := NewTeam(queries).GetTeam(ctx, pgtype.UUID{Valid: true})
		require.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Nil(t, team)
	})
}

func TestGetTeamMissingOrganization(t *testing.T) {
	team, err := NewTeam(nil).GetTeam(t.Context(), pgtype.UUID{Valid: true})
	require.EqualError(t, err, "OrgID missing from context")
	assert.Nil(t, team)
}

func TestListTeams(t *testing.T) {
	queries := teamTestQueries(t)
	org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{
		Name: "List Teams Organization",
		Slug: "list-teams-org",
	})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	service := NewTeam(queries)

	t.Run("empty", func(t *testing.T) {
		teams, err := service.ListTeams(ctx)
		require.NoError(t, err)
		assert.NotNil(t, teams)
		assert.Empty(t, teams)
	})

	platform, err := service.CreateTeam(ctx, TeamParams{
		Name:        "PLATFORM",
		Description: "Platform team"})
	require.NoError(t, err)
	analytics, err := service.CreateTeam(ctx, TeamParams{
		Name:        "ANALYTICS",
		Description: "Analytics team"})

	require.NoError(t, err)
	otherOrg, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name: "Other Organization", Slug: "other-org",
	})
	require.NoError(t, err)
	_, err = queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: otherOrg.ID, Name: "OTHER"})
	require.NoError(t, err)

	teams, err := service.ListTeams(ctx)
	require.NoError(t, err)
	assert.Equal(t, []OrgTeam{
		{ID: analytics.ID, Name: "ANALYTICS", Description: "Analytics team"},
		{ID: platform.ID, Name: "PLATFORM", Description: "Platform team"},
	}, teams)
}

func TestListTeamsMissingOrganization(t *testing.T) {
	teams, err := NewTeam(nil).ListTeams(t.Context())
	require.EqualError(t, err, "OrgID missing from context")
	assert.Nil(t, teams)
}

// Only container setup is shared; each method test creates its own fixtures.
func teamTestQueries(t *testing.T) *sqlc.Queries {
	t.Helper()
	if testing.Short() {
		t.Skip("requires Postgres containers")
	}
	containers, err := testutil.InitializePostgresContainer()
	require.NoError(t, err)
	t.Cleanup(func() {
		containers.Internal.Pool.Close()
		assert.NoError(t, containers.External.DB.Close())
		testutil.TerminateContainer(containers.Internal.Container, containers.External.Container)
	})
	return sqlc.New(containers.Internal.Pool)
}
