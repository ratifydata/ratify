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
			require.Zero(t, count, "draft initialization does not insert columns")
		})
	}
}

func TestActivateContractInvalidBody(t *testing.T) {
	for _, body := range []string{"", "{", "[]", `{"id":"invalid"}`, `{"id":42}`} {
		t.Run(body, func(t *testing.T) {
			rec := httptest.NewRecorder()
			activateContract(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contracts/id/activate", strings.NewReader(body)))
			assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
		})
	}
}

func TestActivateContractMissingIdentity(t *testing.T) {
	for _, missing := range []string{"organization", "user"} {
		t.Run(missing, func(t *testing.T) {
			ctx := t.Context()
			if missing == "user" {
				ctx = context.WithValue(ctx, "OrgID", pgtype.UUID{Bytes: uuid.New(), Valid: true})
			}
			rec := httptest.NewRecorder()
			// An empty service ensures identity checks precede database access.
			activateContract(&contract.ContractApi{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contracts/id/activate", strings.NewReader(`{}`)).WithContext(ctx))
			assertTeamResponse(t, rec, http.StatusInternalServerError, Response{Status: "error", Message: "internal server error"})
		})
	}
}

func TestActivateContractHandler(t *testing.T) {
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
	service := contract.New(containers.Internal.Pool, key)
	handler := activateContract(service)
	remote, err := url.Parse(containers.External.DSN)
	require.NoError(t, err)
	port, err := strconv.Atoi(remote.Port())
	require.NoError(t, err)
	password, _ := remote.User.Password()
	encrypted, err := auth.Encrypt([]byte(key), password)
	require.NoError(t, err)

	for _, name := range []string{"unchanged schema", "changed schema", "already active", "foreign organization", "missing contract", "missing table", "canceled request"} {
		t.Run(name, func(t *testing.T) {
			suffix := uuid.NewString()
			org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Activation handler", Slug: suffix})
			require.NoError(t, err)
			user, err := queries.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: suffix + "@example.com", DisplayName: "Producer", IsActive: true})
			require.NoError(t, err)
			team, err := queries.CreateTeam(t.Context(), sqlc.CreateTeamParams{OrgID: org.ID, Name: "Producers", EmailAddress: suffix + "@example.com"})
			require.NoError(t, err)
			connection, err := queries.CreateDatabaseConnection(t.Context(), sqlc.CreateDatabaseConnectionParams{OrgID: org.ID, DisplayName: "Warehouse", Host: remote.Hostname(), Port: int32(port), DatabaseName: strings.TrimPrefix(remote.Path, "/"), Username: remote.User.Username(), PasswordEncrypted: encrypted.CipherText, Nonce: encrypted.Nonce, SslMode: "disable", Status: "active"})
			require.NoError(t, err)
			table := "activate_" + strings.ReplaceAll(suffix, "-", "")
			_, err = containers.External.DB.ExecContext(t.Context(), `CREATE TABLE public.`+table+` (id integer NOT NULL, note text)`)
			require.NoError(t, err)
			ctx := context.WithValue(t.Context(), "OrgID", org.ID)
			ctx = context.WithValue(ctx, "UserID", user.ID)
			draft, err := service.InitContract(ctx, contract.ContractParams{ConnectionId: connection.ID, TeamId: team.ID, DisplayName: "Orders", SchemaName: "public", TableName: table})
			require.NoError(t, err)
			params := contract.ActivateContract{ID: draft.ContractID}
			wantStatus := http.StatusInternalServerError
			wantResponse := Response{Status: "error", Message: "internal server error"}
			switch name {
			case "unchanged schema", "changed schema":
				wantStatus, wantResponse = http.StatusCreated, Response{Status: "ok"}
				if name == "changed schema" {
					_, err = containers.External.DB.ExecContext(t.Context(), `ALTER TABLE public.`+table+` ALTER COLUMN note SET NOT NULL`)
					require.NoError(t, err)
				}
			case "already active":
				require.NoError(t, service.ActivateContract(ctx, params))
			case "foreign organization":
				other, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Other", Slug: "other-" + suffix})
				require.NoError(t, err)
				ctx = context.WithValue(ctx, "OrgID", other.ID)
			case "missing contract":
				params.ID = pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
			case "missing table":
				_, err = containers.External.DB.ExecContext(t.Context(), `DROP TABLE public.`+table)
				require.NoError(t, err)
				wantStatus = http.StatusNotFound
				wantResponse.Message = "table " + table + " does not exist"
			case "canceled request":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			body, err := json.Marshal(params)
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/contracts/id/activate", bytes.NewReader(body)).WithContext(ctx))
			assertTeamResponse(t, rec, wantStatus, wantResponse)
			stored, err := queries.GetContract(t.Context(), sqlc.GetContractParams{ID: draft.ContractID, OrgID: org.ID})
			require.NoError(t, err)
			wantVersion, wantColumns := int32(1), 0
			wantActive := wantStatus == http.StatusCreated || name == "already active"
			if wantActive {
				require.Equal(t, "active", stored.Status)
			} else {
				require.Equal(t, "draft", stored.Status)
			}
			require.Equal(t, wantActive, stored.ActivatedAt.Valid)
			if name == "changed schema" {
				wantVersion, wantColumns = 2, 2
			}
			require.Equal(t, wantVersion, stored.CurrentVersion)
			versions, err := queries.ListContractVersions(t.Context(), draft.ContractID)
			require.NoError(t, err)
			require.Len(t, versions, int(wantVersion))
			var columns int
			require.NoError(t, containers.Internal.Pool.QueryRow(t.Context(), `SELECT count(*) FROM contract_columns WHERE contract_version_id=$1`, versions[0].ID).Scan(&columns))
			require.Equal(t, wantColumns, columns)
		})
	}
}
