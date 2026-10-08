package access

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/apperrors"

	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/stretchr/testify/require"
)

func TestAddTeamMember(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Members", Slug: "members"})
	require.NoError(t, err)
	team, err := q.CreateTeam(t.Context(), sqlc.CreateTeamParams{OrgID: org.ID, Name: "TEAM", EmailAddress: "team@example.com"})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	service := NewMember(q)
	t.Run("new user", func(t *testing.T) {
		member, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "  new@example.com  "})
		require.NoError(t, err)
		user, err := q.GetUser(ctx, member.UserID)
		require.NoError(t, err)
		require.Equal(t, org.ID, user.OrgID)
		require.Equal(t, "new@example.com", user.DisplayName)
		require.Equal(t, "new@example.com", user.Email)
		require.True(t, user.IsActive)
	})
	t.Run("existing user and duplicate", func(t *testing.T) {
		user, err := q.CreateUser(ctx, sqlc.CreateUserParams{OrgID: org.ID, Email: "existing@example.com", DisplayName: "Original", IsActive: true})
		require.NoError(t, err)
		added, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: user.Email})
		require.NoError(t, err)
		require.Equal(t, user.ID, added.UserID)
		_, err = service.AddTeamMember(ctx, team.ID, MemberParams{Email: user.Email})
		require.EqualError(t, err, "team member already exists")
	})
	t.Run("concurrent duplicate", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for range 2 {
			wg.Go(func() {
				_, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "race@example.com"})
				errs <- err
			})
		}
		wg.Wait()
		close(errs)
		success, duplicate := 0, 0
		for err := range errs {
			if err == nil {
				success++
			} else {
				require.Error(t, err)
				duplicate++
			}
		}
		require.Equal(t, 1, success)
		require.Equal(t, 1, duplicate)
	})
	t.Run("invalid email", func(t *testing.T) {
		_, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "bad"})
		require.EqualError(t, err, "valid email is required")
	})
}

func TestRemoveTeamMember(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Remove", Slug: "remove"})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	team, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: org.ID, Name: "TEAM", EmailAddress: "team@example.com"})
	require.NoError(t, err)
	service := NewMember(q)
	member, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "member@example.com"})
	require.NoError(t, err)
	require.NoError(t, service.RemoveTeamMember(ctx, team.ID, member.UserID))
	stored, err := q.GetTeamMember(ctx, sqlc.GetTeamMemberParams{TeamID: team.ID, UserID: member.UserID})
	require.NoError(t, err)
	require.True(t, stored.DeletedAt.Valid)

	_, err = q.GetUser(ctx, member.UserID)
	require.NoError(t, err)
	require.NoError(t, service.RemoveTeamMember(ctx, team.ID, member.UserID))
	members, err := service.ListTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, members)
}

func TestListTeamMembers(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "List", Slug: "list"})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	team, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: org.ID, Name: "TEAM", EmailAddress: "team@example.com"})
	require.NoError(t, err)
	service := NewMember(q)
	members, err := service.ListTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.NotNil(t, members)
	require.Empty(t, members)
	added, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "member@example.com"})
	require.NoError(t, err)
	members, err = service.ListTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.Equal(t, []TeamMember{{TeamID: team.ID, UserID: added.UserID, Role: "member", Email: "member@example.com"}}, members)
}

func TestTeamMemberOrganizationIsolation(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Owner", Slug: "owner"})
	require.NoError(t, err)
	other, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Other", Slug: "other"})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	foreign := context.WithValue(t.Context(), "OrgID", other.ID)
	team, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: org.ID, Name: "TEAM", EmailAddress: "team@example.com"})
	require.NoError(t, err)
	service := NewMember(q)
	member, err := service.AddTeamMember(ctx, team.ID, MemberParams{Email: "member@example.com"})
	require.NoError(t, err)
	_, err = service.AddTeamMember(foreign, team.ID, MemberParams{Email: "intruder@example.com"})
	require.ErrorAs(t, err, new(*apperrors.NotFoundError))
	exists, err := q.CheckUserExistByEmail(foreign, sqlc.CheckUserExistByEmailParams{OrgID: other.ID, Email: "intruder@example.com"})
	require.NoError(t, err)
	require.False(t, exists)
	_, err = service.ListTeamMembers(foreign, team.ID)
	require.ErrorAs(t, err, new(*apperrors.NotFoundError))
	require.ErrorAs(t, service.RemoveTeamMember(foreign, team.ID, member.UserID), new(*apperrors.NotFoundError))
	members, err := service.ListTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.ErrorAs(t, service.RemoveTeamMember(ctx, team.ID, pgtype.UUID{Valid: true}), new(*apperrors.NotFoundError))
}

func TestTeamMemberMissingOrganization(t *testing.T) {
	service := NewMember(nil)
	id := pgtype.UUID{Valid: true}
	_, err := service.AddTeamMember(t.Context(), id, MemberParams{Email: "member@example.com"})
	require.ErrorAs(t, err, new(*apperrors.BadRequestError))
	_, err = service.ListTeamMembers(t.Context(), id)
	require.ErrorAs(t, err, new(*apperrors.BadRequestError))
	require.ErrorAs(t, service.RemoveTeamMember(t.Context(), id, id), new(*apperrors.BadRequestError))
}
