package sqlc

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueries_CreateOrganization(t *testing.T) {
	arg := CreateOrganizationParams{
		Name:                  "OrgTest",
		Slug:                  "orgTest",
		SmtpHost:              pgtype.Text{String: "http://host.com"},
		SmtpFromAddress:       pgtype.Text{String: "address1"},
		SmtpUsername:          pgtype.Text{String: "username"},
		SmtpPasswordEncrypted: pgtype.Text{String: "encPassword"},
		AutoApproveAdditive:   true,
	}
	queries := New(testDB.Pool)
	org, err := queries.CreateOrganization(context.Background(), arg)

	require.NoError(t, err)
	require.NotNil(t, org.ID)
	assert.Equal(t, arg.Name, org.Name)
	assert.Equal(t, arg.Slug, org.Slug)
}

func TestQueries_CreateOrganization_Error(t *testing.T) {
	queries := New(testDB.Pool)
	ctx := context.Background()
	arg := CreateOrganizationParams{
		Name: "Duplicate Organization",
		Slug: "duplicate-organization",
	}

	_, err := queries.CreateOrganization(ctx, arg)
	require.NoError(t, err)

	arg.Name = "Another Organization"
	_, err = queries.CreateOrganization(ctx, arg)

	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
	assert.Equal(t, "organizations_slug_key", pgErr.ConstraintName)
}
