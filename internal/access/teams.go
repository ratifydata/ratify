package access

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type TeamParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OrgTeam struct {
	ID          pgtype.UUID `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
}

type Team struct {
	db *sqlc.Queries
}

func NewTeam(db *sqlc.Queries) *Team {
	return &Team{db: db}
}

func (t *Team) CreateTeam(ctx context.Context, args TeamParams) (*OrgTeam, error) {
	orgID, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok || !orgID.Valid {
		slog.Error("OrgID missing from context")
		return nil, fmt.Errorf("OrgID missing from context")
	}
	teamExists, err := t.db.GetTeamByName(ctx, sqlc.GetTeamByNameParams{
		Name:  formatString(args.Name),
		OrgID: orgID,
	})
	if err != nil {
		slog.Error("Error checking if team exists")
		return nil, err
	}
	if teamExists {
		slog.Error("Team already exists")
		return nil, fmt.Errorf("team already exists")
	}

	team, err := t.db.CreateTeam(ctx, sqlc.CreateTeamParams{
		Name:  formatString(args.Name),
		OrgID: orgID,
		Description: pgtype.Text{
			String: args.Description,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, err
	}

	return &OrgTeam{
		ID:          team.ID,
		Name:        team.Name,
		Description: team.Description.String,
	}, nil

}

func (t *Team) GetTeam(ctx context.Context, teamId pgtype.UUID) (*OrgTeam, error) {
	orgID, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok || !orgID.Valid {
		slog.Error("OrgID missing from context")
		return nil, fmt.Errorf("OrgID missing from context")
	}

	team, err := t.db.GetTeam(ctx, teamId)
	if err != nil {
		return nil, err
	}
	return &OrgTeam{
		ID:          team.ID,
		Name:        team.Name,
		Description: team.Description.String,
	}, nil

}

func (t *Team) ListTeams(ctx context.Context) ([]OrgTeam, error) {
	orgID, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok || !orgID.Valid {
		slog.Error("OrgID missing from context")
		return nil, fmt.Errorf("OrgID missing from context")
	}
	teams, err := t.db.ListTeamsByOrg(ctx, orgID)
	if err != nil {
		slog.Error("Error listing teams")
		return nil, err
	}
	allTeams := make([]OrgTeam, 0, len(teams))
	for _, team := range teams {
		allTeams = append(allTeams, OrgTeam{
			ID:          team.ID,
			Name:        team.Name,
			Description: team.Description.String,
		})
	}

	return allTeams, nil
}

func formatString(value string) string {
	return strings.ToUpper(strings.ReplaceAll(value, " ", ""))
}
