package contract

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/auth"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/ratifydata/ratify/internal/testutil"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

var testDB *testutil.TestContainer

func TestMain(m *testing.M) {
	var err error
	testDB, err = testutil.InitializePostgresContainer()
	if err != nil {
		os.Exit(1)
	}
	code := m.Run()
	defer func(DB *sql.DB) {
		err := DB.Close()
		if err != nil {
			slog.Error("failed to close DB container")
		}
	}(testDB.External.DB)
	testDB.Internal.Pool.Close()
	testutil.TerminateContainer(testDB.Internal.Container, testDB.External.Container)
	os.Exit(code)
}

type contractFixture struct {
	api           *ContractApi
	queries       *sqlc.Queries
	ctx           context.Context
	orgID, userID pgtype.UUID
	args          ContractParams
	snapshot      *schema.TableSnapshot
}

func newContractFixture(t *testing.T) contractFixture {
	t.Helper()
	q := sqlc.New(testDB.Internal.Pool)
	suffix := uuid.NewString()
	org, err := q.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: "Contracts", Slug: suffix})
	require.NoError(t, err)
	team, err := q.CreateTeam(t.Context(), sqlc.CreateTeamParams{OrgID: org.ID, Name: "Producers", EmailAddress: suffix + "@example.com"})
	require.NoError(t, err)
	user, err := q.CreateUser(t.Context(), sqlc.CreateUserParams{OrgID: org.ID, Email: suffix + "@example.com", DisplayName: "Producer", IsActive: true})
	require.NoError(t, err)
	remote, err := url.Parse(testDB.External.DSN)
	require.NoError(t, err)
	port, err := strconv.Atoi(remote.Port())
	require.NoError(t, err)
	password, _ := remote.User.Password()
	encrypted, err := auth.Encrypt([]byte(testEncryptionKey), password)
	require.NoError(t, err)
	connection, err := q.CreateDatabaseConnection(t.Context(), sqlc.CreateDatabaseConnectionParams{
		OrgID: org.ID, DisplayName: "Warehouse", Host: remote.Hostname(), Port: int32(port), DatabaseName: remote.Path[1:], Username: remote.User.Username(), PasswordEncrypted: encrypted.CipherText, Nonce: encrypted.Nonce, SslMode: "disable", Status: "active",
	})
	require.NoError(t, err)
	schemaName := "contract_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = testDB.External.DB.ExecContext(t.Context(), `CREATE SCHEMA `+schemaName+`; CREATE TABLE `+schemaName+`.orders (tenant_id integer, id integer, note text, PRIMARY KEY (tenant_id, id))`)
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), "OrgID", org.ID)
	ctx = context.WithValue(ctx, "UserID", user.ID)
	return contractFixture{
		api: New(testDB.Internal.Pool, testEncryptionKey), queries: q, ctx: ctx, orgID: org.ID, userID: user.ID,
		args: ContractParams{ConnectionId: connection.ID, TeamId: team.ID, DisplayName: "Orders contract", SchemaName: schemaName, TableName: "orders", Description: pgtype.Text{String: "Initial snapshot", Valid: true}},
		snapshot: &schema.TableSnapshot{SchemaName: schemaName, TableName: "orders", Columns: []schema.TableData{
			{ColumnName: "tenant_id", DataType: "integer", IsPrimaryKey: true, OrdinalPosition: 1},
			{ColumnName: "id", DataType: "integer", IsPrimaryKey: true, OrdinalPosition: 2},
			{ColumnName: "note", DataType: "text", IsNullable: true, OrdinalPosition: 3},
		}},
	}
}

func assertNoContracts(t *testing.T, f contractFixture) {
	t.Helper()
	var contracts, versions, columns int
	require.NoError(t, testDB.Internal.Pool.QueryRow(t.Context(), `SELECT
 (SELECT count(*) FROM contracts WHERE org_id=$1),
 (SELECT count(*) FROM contract_versions v JOIN contracts c ON c.id=v.contract_id WHERE c.org_id=$1),
 (SELECT count(*) FROM contract_columns cc JOIN contract_versions v ON v.id=cc.contract_version_id JOIN contracts c ON c.id=v.contract_id WHERE c.org_id=$1)`, f.orgID).Scan(&contracts, &versions, &columns))
	require.Zero(t, contracts)
	require.Zero(t, versions)
	require.Zero(t, columns)
}

func TestInitContract(t *testing.T) {
	for _, tt := range []struct {
		name string
		sla  *int32
	}{
		{name: "omitted SLA"}, {name: "zero SLA", sla: new(int32)}, {name: "positive SLA", sla: func() *int32 { v := int32(12); return &v }()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newContractFixture(t)
			f.args.FreshnessSlaHours = tt.sla
			result, err := f.api.InitContract(f.ctx, f.args)
			require.NoError(t, err)
			require.True(t, result.ContractID.Valid)
			require.Equal(t, int32(1), result.CurrentVersion)
			require.Equal(t, "draft", result.Status)
			require.Equal(t, f.snapshot.Columns, result.TableColumns)
			stored, err := f.queries.GetContract(f.ctx, sqlc.GetContractParams{ID: result.ContractID, OrgID: f.orgID})
			require.NoError(t, err)
			require.Equal(t, f.orgID, stored.OrgID)
			require.Equal(t, f.args.TeamId, stored.ProducerTeamID)
			require.Equal(t, f.args.ConnectionId, stored.ConnectionID)
			require.Equal(t, f.args.DisplayName, stored.DisplayName)
			require.Equal(t, f.args.SchemaName, stored.SchemaName)
			require.Equal(t, f.args.TableName, stored.TableName)
			require.Equal(t, "draft", stored.Status)
			require.Equal(t, result.CurrentVersion, stored.CurrentVersion)
			require.Equal(t, tt.sla != nil, stored.FreshnessSlaHours.Valid)
			if tt.sla != nil {
				require.Equal(t, *tt.sla, stored.FreshnessSlaHours.Int32)
			}
			version, err := f.queries.GetContractVersionByNumber(f.ctx, sqlc.GetContractVersionByNumberParams{ContractID: stored.ID, VersionNumber: 1})
			require.NoError(t, err)
			require.Equal(t, f.userID, version.CreatedBy)
			require.Equal(t, f.args.Description, version.ChangeSummary)
			var snapshot schema.TableSnapshot
			require.NoError(t, json.Unmarshal(version.SchemaSnapshot, &snapshot))
			require.Equal(t, f.snapshot, &snapshot)
			rows, err := testDB.Internal.Pool.Query(f.ctx, `SELECT column_name, data_type, is_nullable, is_primary_key, ordinal_position FROM contract_columns WHERE contract_version_id=$1 ORDER BY ordinal_position`, version.ID)
			require.NoError(t, err)
			defer rows.Close()
			var columns []schema.TableData
			for rows.Next() {
				var col schema.TableData
				require.NoError(t, rows.Scan(&col.ColumnName, &col.DataType, &col.IsNullable, &col.IsPrimaryKey, &col.OrdinalPosition))
				columns = append(columns, col)
			}
			require.NoError(t, rows.Err())
			require.Empty(t, columns, "draft initialization does not insert columns")
			data, err := json.Marshal(result)
			require.NoError(t, err)
			var response map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &response))
			require.Contains(t, response, "contract_id")
			require.NotContains(t, response, "contact_id")
		})
	}
}

func TestInitContractRollback(t *testing.T) {
	for _, stage := range []string{"contract", "version"} {
		t.Run(stage, func(t *testing.T) {
			f := newContractFixture(t)
			switch stage {
			case "contract":
				f.args.ConnectionId = pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}
			case "version":
				f.userID = pgtype.UUID{Bytes: [16]byte(uuid.New()), Valid: true}

			}
			result, err := f.api.initContractSnapshot(f.ctx, f.orgID, f.userID, f.args, f.snapshot)
			require.Error(t, err)
			require.Nil(t, result)
			assertNoContracts(t, f)
		})
	}
}

func TestInitContractRejectsForeignResources(t *testing.T) {
	for _, resource := range []string{"team", "connection"} {
		t.Run(resource, func(t *testing.T) {
			f, other := newContractFixture(t), newContractFixture(t)
			if resource == "team" {
				f.args.TeamId = other.args.TeamId
			} else {
				f.args.ConnectionId = other.args.ConnectionId
			}
			result, err := f.api.InitContract(f.ctx, f.args)
			require.Error(t, err)
			require.Nil(t, result)
			assertNoContracts(t, f)
		})
	}
}

func TestContractVersionResponse(t *testing.T) {
	f := newContractFixture(t)
	contract, err := f.api.InitContract(f.ctx, f.args)
	require.NoError(t, err)
	description := pgtype.Text{String: "Updated metadata", Valid: true}
	version, err := f.api.ContractVersion(f.ctx, ContractVersion{ContractId: contract.ContractID, Version: 2, User: f.userID, Description: description, Snapshot: f.snapshot})
	require.NoError(t, err)
	stored, err := f.queries.GetContractVersion(f.ctx, version.ID)
	require.NoError(t, err)
	require.Equal(t, &ContractVersionModel{ID: stored.ID, ContractId: stored.ContractID, Version: stored.VersionNumber, User: stored.CreatedBy, Description: stored.ChangeSummary}, version)
}

func TestInitContractRejectsInvalidIdentity(t *testing.T) {
	f := newContractFixture(t)
	for _, tt := range []struct {
		name      string
		org, user interface{}
	}{
		{"missing org", nil, f.userID}, {"invalid org", pgtype.UUID{}, f.userID}, {"wrong org type", "org", f.userID},
		{"missing user", f.orgID, nil}, {"invalid user", f.orgID, pgtype.UUID{}}, {"wrong user type", f.orgID, "user"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.org != nil {
				ctx = context.WithValue(ctx, "OrgID", tt.org)
			}
			if tt.user != nil {
				ctx = context.WithValue(ctx, "UserID", tt.user)
			}
			result, err := f.api.InitContract(ctx, f.args)
			require.Error(t, err)
			require.Nil(t, result)
			assertNoContracts(t, f)
		})
	}
}

func TestInitContractCanceledRequest(t *testing.T) {
	f := newContractFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	result, err := f.api.InitContract(ctx, f.args)
	require.Error(t, err)
	require.Nil(t, result)
	assertNoContracts(t, f)
}

func TestActivateContract(t *testing.T) {
	for _, changed := range []bool{false, true} {
		name := "unchanged schema"
		if changed {
			name = "changed schema"
		}
		t.Run(name, func(t *testing.T) {
			f := newContractFixture(t)
			draft, err := f.api.InitContract(f.ctx, f.args)
			require.NoError(t, err)
			if changed {
				_, err = testDB.External.DB.ExecContext(f.ctx, `ALTER TABLE `+f.args.SchemaName+`.orders ALTER COLUMN note SET NOT NULL`)
				require.NoError(t, err)
			}
			require.NoError(t, f.api.ActivateContract(f.ctx, ActivateContract{ID: draft.ContractID}))
			stored, err := f.queries.GetContract(f.ctx, sqlc.GetContractParams{ID: draft.ContractID, OrgID: f.orgID})
			require.NoError(t, err)
			require.Equal(t, "active", stored.Status)
			require.True(t, stored.ActivatedAt.Valid)
			wantVersion, wantColumns := int32(1), 0
			if changed {
				wantVersion, wantColumns = 2, len(f.snapshot.Columns)
			}
			require.Equal(t, wantVersion, stored.CurrentVersion)
			versions, err := f.queries.ListContractVersions(f.ctx, draft.ContractID)
			require.NoError(t, err)
			require.Len(t, versions, int(wantVersion))
			var columns int
			require.NoError(t, testDB.Internal.Pool.QueryRow(f.ctx, `SELECT count(*) FROM contract_columns WHERE contract_version_id=$1`, versions[0].ID).Scan(&columns))
			require.Equal(t, wantColumns, columns)
			require.Error(t, f.api.ActivateContract(f.ctx, ActivateContract{ID: draft.ContractID}))
		})
	}
}

func TestActivateContractConcurrent(t *testing.T) {
	f := newContractFixture(t)
	draft, err := f.api.InitContract(f.ctx, f.args)
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- f.api.ActivateContract(f.ctx, ActivateContract{ID: draft.ContractID})
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if <-results == nil {
			successes++
		}
	}
	require.Equal(t, 1, successes, "only one concurrent activation may succeed")
}

func TestActivateContractRollback(t *testing.T) {
	f := newContractFixture(t)
	first, err := f.api.InitContract(f.ctx, f.args)
	require.NoError(t, err)
	require.NoError(t, f.api.ActivateContract(f.ctx, ActivateContract{ID: first.ContractID}))
	second, err := f.api.InitContract(f.ctx, f.args)
	require.NoError(t, err)
	_, err = testDB.External.DB.ExecContext(f.ctx, `ALTER TABLE `+f.args.SchemaName+`.orders ALTER COLUMN note SET NOT NULL`)
	require.NoError(t, err)
	// The existing active contract makes the final update fail its unique index.
	require.Error(t, f.api.ActivateContract(f.ctx, ActivateContract{ID: second.ContractID}))
	stored, err := f.queries.GetContract(f.ctx, sqlc.GetContractParams{ID: second.ContractID, OrgID: f.orgID})
	require.NoError(t, err)
	require.Equal(t, "draft", stored.Status)
	require.Equal(t, int32(1), stored.CurrentVersion)
	require.False(t, stored.ActivatedAt.Valid)
	versions, err := f.queries.ListContractVersions(f.ctx, second.ContractID)
	require.NoError(t, err)
	require.Len(t, versions, 1, "new version must roll back when activation fails")
}
