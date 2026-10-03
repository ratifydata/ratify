package access

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/util"
)

var (
	ErrMemberEmail        = errors.New("valid email is required")
	ErrMemberExists       = errors.New("team member already exists")
	ErrMemberNotFound     = errors.New("team member not found")
	ErrTeamNotFound       = errors.New("team not found")
	ErrMemberOrganization = errors.New("OrgID missing from context")
)

type MemberParams struct {
	Email string `json:"email"`
}

type TeamMember struct {
	TeamID pgtype.UUID `json:"team_id"`
	UserID pgtype.UUID `json:"user_id"`
	Role   string      `json:"role"`
	Email  string      `json:"email"`
}

type Member struct {
	user *User
	db   *sqlc.Queries
}

func NewMember(db *sqlc.Queries) *Member {
	return &Member{
		db:   db,
		user: NewUser(db),
	}
}

func (m *Member) AddTeamMember(ctx context.Context, teamID pgtype.UUID, params MemberParams) (*TeamMember, error) {
	orgId, err := util.ValidateOrgId(ctx)
	if err != nil {
		return nil, err
	}
	email := strings.TrimSpace(params.Email)
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		slog.Error("failed to parse email address")
		return nil, ErrMemberEmail
	}

	user, err := m.user.GetUserViaMembership(ctx, orgId, UserParams{
		Email:       email,
		DisplayName: email,
	})
	if err != nil {
		return nil, err
	}
	//Check if team member exists
	teamMemberExist, err := m.db.TeamMemberExist(ctx, sqlc.TeamMemberExistParams{
		TeamID: teamID,
		UserID: user.ID,
	})
	if err != nil {
		slog.Error("Failed to get team member")
		return nil, err
	}
	if teamMemberExist {
		return nil, ErrMemberExists
	}
	//Link team member after passing all checks
	teamMember, err := m.db.CreateTeamMember(ctx, sqlc.CreateTeamMemberParams{
		TeamID: teamID,
		UserID: user.ID,
		Role:   "member",
	})

	if err != nil {
		slog.Error("Failed to create team member")
		return nil, err
	}
	return &TeamMember{
		TeamID: teamMember.TeamID,
		UserID: user.ID,
		Role:   teamMember.Role}, nil
}

func (m *Member) RemoveTeamMember(ctx context.Context, teamID, userID pgtype.UUID) error {
	_, err := util.ValidateOrgId(ctx)
	if err != nil {
		return nil
	}
	_, err = m.db.RemoveTeamMember(ctx, sqlc.RemoveTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *Member) ListTeamMembers(ctx context.Context, teamID pgtype.UUID) ([]TeamMember, error) {
	_, err := util.ValidateOrgId(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := m.db.ListTeamMembersByTeam(ctx, teamID)
	if err != nil {
		slog.Error("Failed to list team members")
		return nil, err
	}
	members := make([]TeamMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, TeamMember{TeamID: row.TeamID, UserID: row.UserID, Role: row.Role, Email: row.Email})
	}
	return members, nil
}
