package access

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/stretchr/testify/require"
)

func TestCreateUserViaEmail(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Users", Slug: "users"})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	service := NewUser(q)
	t.Run("create", func(t *testing.T) {
		got, err := service.CreateUserViaEmail(ctx, UserParams{Email: "create@example.com", DisplayName: "New User"})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "create@example.com", got.Email)
		require.Equal(t, "New User", got.DisplayName)
		require.True(t, got.Active)
		stored, err := q.GetUser(ctx, got.ID)
		require.NoError(t, err)
		require.Equal(t, org.ID, stored.OrgID)
		require.Equal(t, got.Email, stored.Email)
		require.Equal(t, got.DisplayName, stored.DisplayName)
		require.True(t, stored.IsActive)
	})
	t.Run("duplicate", func(t *testing.T) {
		original, err := q.CreateUser(ctx, sqlc.CreateUserParams{OrgID: org.ID, Email: "duplicate@example.com", DisplayName: "Original", IsActive: true})
		require.NoError(t, err)
		got, err := service.CreateUserViaEmail(ctx, UserParams{Email: original.Email, DisplayName: "Replacement"})
		require.EqualError(t, err, "user with email duplicate@example.com already exists")
		require.Nil(t, got)
		stored, err := q.GetUser(ctx, original.ID)
		require.NoError(t, err)
		require.Equal(t, original, stored)
	})
	t.Run("canceled query", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		got, err := service.CreateUserViaEmail(canceled, UserParams{Email: "cancel@example.com"})
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, got)
	})
}

func TestCreateUserViaEmailInvalidOrganization(t *testing.T) {
	service := NewUser(nil)
	got, err := service.CreateUserViaEmail(t.Context(), UserParams{Email: "user@example.com"})
	require.EqualError(t, err, "OrgID missing from context")
	require.Nil(t, got)
	ctx := context.WithValue(t.Context(), "OrgID", pgtype.UUID{})
	got, err = service.CreateUserViaEmail(ctx, UserParams{Email: "user@example.com"})
	require.EqualError(t, err, "OrgID not valid")
	require.Nil(t, got)
}

func TestGetUserViaMembership(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Membership", Slug: "membership"})
	require.NoError(t, err)
	service := NewUser(q)
	t.Run("new user", func(t *testing.T) {
		got, err := service.GetUserViaMembership(t.Context(), org.ID, UserParams{Email: "new@example.com", DisplayName: "New"})
		require.NoError(t, err)
		require.NotNil(t, got)
		stored, err := q.GetUser(t.Context(), got.ID)
		require.NoError(t, err)
		require.Equal(t, org.ID, stored.OrgID)
		require.Equal(t, &UserDetails{ID: stored.ID, Email: "new@example.com", DisplayName: "New", Active: true}, got)
	})
	t.Run("existing user", func(t *testing.T) {
		original, err := q.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: "existing@example.com", DisplayName: "Original", IsActive: true})
		require.NoError(t, err)
		got, err := service.GetUserViaMembership(t.Context(), org.ID, UserParams{Email: original.Email, DisplayName: "Replacement"})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, original.ID, got.ID)
		require.Equal(t, original.Email, got.Email)
		stored, err := q.GetUser(t.Context(), original.ID)
		require.NoError(t, err)
		require.Equal(t, original, stored)
	})
	t.Run("same email in different organizations", func(t *testing.T) {
		other, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Other", Slug: "other-member"})
		require.NoError(t, err)
		first, err := service.GetUserViaMembership(t.Context(), org.ID, UserParams{Email: "shared@example.com"})
		require.NoError(t, err)
		second, err := service.GetUserViaMembership(t.Context(), other.ID, UserParams{Email: "shared@example.com"})
		require.NoError(t, err)
		require.NotEqual(t, first.ID, second.ID)
		again, err := service.GetUserViaMembership(t.Context(), other.ID, UserParams{Email: "shared@example.com"})
		require.NoError(t, err)
		require.Equal(t, second.ID, again.ID)
	})
	t.Run("canceled query", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		got, err := service.GetUserViaMembership(ctx, org.ID, UserParams{Email: "cancel@example.com"})
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, got)
	})
	t.Run("unknown organization", func(t *testing.T) {
		got, err := service.GetUserViaMembership(t.Context(), pgtype.UUID{Valid: true}, UserParams{Email: "orphan@example.com"})
		require.Error(t, err)
		require.Nil(t, got)
		exists, err := q.CheckUserExistByEmail(t.Context(), sqlc.CheckUserExistByEmailParams{Email: "orphan@example.com", OrgID: pgtype.UUID{Valid: true}})
		require.NoError(t, err)
		require.False(t, exists)
	})
}

func TestUserExists(t *testing.T) {
	q := teamTestQueries(t)
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Exists", Slug: "exists"})
	require.NoError(t, err)
	_, err = q.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: "exists@example.com", DisplayName: "Existing", IsActive: true})
	require.NoError(t, err)
	service := NewUser(q)
	exists, err := service.userExists(t.Context(), "exists@example.com", org.ID)
	require.NoError(t, err)
	require.True(t, exists)
	exists, err = service.userExists(t.Context(), "missing@example.com", org.ID)
	require.NoError(t, err)
	require.False(t, exists)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	exists, err = service.userExists(ctx, "exists@example.com", org.ID)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, exists)
}
