package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/contract"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/ratifydata/ratify/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCreateContractInvalidBody(t *testing.T) {
	for _, body := range []string{"", "{", "[]", `{"connection_id":"invalid"}`, `{"freshness_sla_hours":"invalid"}`} {
		t.Run(body, func(t *testing.T) {
			rec := httptest.NewRecorder()
			// A nil service ensures decoding errors return before invoking contract logic.
			createContract(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contracts", strings.NewReader(body)))
			assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
		})
	}
}

func TestCreateContractHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Postgres containers")
	}
	containers, err := testutil.InitializePostgresContainer()
	require.NoError(t, err)
	t.Cleanup(func() {
		containers.Internal.Pool.Close()
		require.NoError(t, containers.External.DB.Close())
		testutil.TerminateContainer(containers.Internal.Container, containers.External.Container)
	})
	const key = "0123456789abcdef0123456789abcdef"
	queries := sqlc.New(containers.Internal.Pool)
	handler := createContract(contract.New(containers.Internal.Pool, key))
	remote, err := url.Parse(containers.External.DSN)
	require.NoError(t, err)
	port, err := strconv.Atoi(remote.Port())
	require.NoError(t, err)
	password, _ := remote.User.Password()
	encrypted, err := auth.Encrypt([]byte(key), password)
	require.NoError(t, err)

	for _, name := range []string{"success", "missing table", "foreign connection", "foreign team", "missing organization", "missing user", "canceled request"} {
		t.Run(name, func(t *testing.T) {
			suffix := uuid.NewString()
			org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Contract handler", Slug: suffix})
			require.NoError(t, err)
			user, err := queries.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: suffix + "@example.com", DisplayName: "Producer", IsActive: true})
			require.NoError(t, err)
			team, err := queries.CreateTeam(t.Context(), sqlc.CreateTeamParams{OrgID: org.ID, Name: "Producers", EmailAddress: suffix + "@example.com"})
			require.NoError(t, err)
			connection, err := queries.CreateDatabaseConnection(t.Context(), sqlc.CreateDatabaseConnectionParams{OrgID: org.ID, DisplayName: "Warehouse", Host: remote.Hostname(), Port: int32(port), DatabaseName: strings.TrimPrefix(remote.Path, "/"), Username: remote.User.Username(), PasswordEncrypted: encrypted.CipherText, Nonce: encrypted.Nonce, SslMode: "disable", Status: "active"})
			require.NoError(t, err)
			sla := int32(12)
			params := contract.ContractParams{ConnectionId: connection.ID, TeamId: team.ID, DisplayName: "Orders contract", SchemaName: "public", TableName: "test_table_1", Description: pgtype.Text{String: "Handler test", Valid: true}, FreshnessSlaHours: &sla}
			ctx := context.WithValue(t.Context(), "OrgID", org.ID)
			ctx = context.WithValue(ctx, "UserID", user.ID)
			status := http.StatusInternalServerError
			message := "internal server error"
			switch name {
			case "success":
				status = http.StatusCreated
			case "missing table":
				params.TableName = "missing_" + strings.ReplaceAll(suffix, "-", "")
				status = http.StatusNotFound
				message = "table " + params.TableName + " does not exist"
			case "foreign connection", "foreign team":
				other, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Other", Slug: "other-" + suffix})
				require.NoError(t, err)
				if name == "foreign connection" {
					// Use a real connection belonging to a different organization.
					_, err := containers.Internal.Pool.Exec(t.Context(), `UPDATE database_connections SET org_id=$1 WHERE id=$2`, other.ID, connection.ID)
					require.NoError(t, err)
				} else {
					_, err := containers.Internal.Pool.Exec(t.Context(), `UPDATE teams SET org_id=$1 WHERE id=$2`, other.ID, team.ID)
					require.NoError(t, err)
				}
			case "missing organization":
				ctx = context.WithValue(t.Context(), "UserID", user.ID)
			case "missing user":
				ctx = context.WithValue(t.Context(), "OrgID", org.ID)
			case "canceled request":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			body, err := json.Marshal(params)
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contracts", bytes.NewReader(body)).WithContext(ctx))
			stored, err := queries.ListContractsByOrg(t.Context(), org.ID)
			require.NoError(t, err)
			if name != "success" {
				assertTeamResponse(t, rec, status, Response{Status: "error", Message: message})
				require.Empty(t, stored)
				return
			}
			require.Len(t, stored, 1)
			require.Equal(t, params.SchemaName, stored[0].SchemaName)
			require.Equal(t, params.TableName, stored[0].TableName)
			require.Equal(t, team.ID, stored[0].ProducerTeamID)
			require.Equal(t, pgtype.Int4{Int32: sla, Valid: true}, stored[0].FreshnessSlaHours)
			version, err := queries.GetContractVersionByNumber(t.Context(), sqlc.GetContractVersionByNumberParams{ContractID: stored[0].ID, VersionNumber: 1})
			require.NoError(t, err)
			require.Equal(t, user.ID, version.CreatedBy)
			require.Equal(t, params.Description, version.ChangeSummary)
			snapshot, err := schema.GetTableSchema(t.Context(), containers.External.DB, params.SchemaName, params.TableName)
			require.NoError(t, err)
			assertTeamResponse(t, rec, status, Response{Status: "ok", Body: contract.InitContractModel{ContractID: stored[0].ID, CurrentVersion: stored[0].CurrentVersion, Status: "draft", TableName: params.TableName, TableColumns: snapshot.Columns}})
			require.Equal(t, int32(1), stored[0].CurrentVersion)
			var count int
			require.NoError(t, containers.Internal.Pool.QueryRow(t.Context(), `SELECT count(*) FROM contract_columns WHERE contract_version_id=$1`, version.ID).Scan(&count))
			require.Equal(t, len(snapshot.Columns), count)
		})
	}
}
