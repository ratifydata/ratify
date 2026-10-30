package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/ratifydata/ratify/internal/util"
)

type contractDB interface {
	sqlc.DBTX
	Begin(context.Context) (pgx.Tx, error)
}

type ContractApi struct {
	transactions contractDB
	db           *sqlc.Queries
	inspection   *schema.Inspector
}

type ContractParams struct {
	ConnectionId      pgtype.UUID `json:"connection_id"`
	TeamId            pgtype.UUID `json:"team_id"`
	DisplayName       string      `json:"display_name"`
	Description       pgtype.Text `json:"description"`
	FreshnessSlaHours *int32      `json:"freshness_sla_hours"`
	SchemaName        string      `json:"schema_name"`
	TableName         string      `json:"table_name"`
}

type ContractVersion struct {
	ContractId  pgtype.UUID           `json:"contract_id"`
	Version     int32                 `json:"version"`
	User        pgtype.UUID           `json:"user"`
	Description pgtype.Text           `json:"description"`
	Snapshot    *schema.TableSnapshot `json:"snapshot"`
}

type ContractColumn struct {
	ContractVersionId pgtype.UUID `json:"contract_version_id"`
	ColumnName        string      `json:"column_name"`
	DataType          string      `json:"data_type"`
	IsNullable        bool        `json:"is_nullable"`
	IsPrimaryKey      bool        `json:"is_primary_key"`
	Description       pgtype.Text `json:"description"`
	Constraints       []byte      `json:"constraints"`
	Position          int32       `json:"position"`
}

type ContractModel struct {
	ID                pgtype.UUID
	ConnectionID      pgtype.UUID
	ProducerTeamID    pgtype.UUID
	DisplayName       string
	SchemaName        string
	TableName         string
	Description       pgtype.Text
	Status            string
	FreshnessSlaHours pgtype.Int4
}

type ActivateContract struct {
	ID pgtype.UUID `json:"id"`
}

type ContractVersionModel struct {
	ID          pgtype.UUID `json:"contract_version_id"`
	ContractId  pgtype.UUID `json:"contract_id"`
	Version     int32       `json:"version"`
	User        pgtype.UUID `json:"user"`
	Description pgtype.Text `json:"description"`
}

type InitContractModel struct {
	ContractID     pgtype.UUID        `json:"contract_id"`
	CurrentVersion int32              `json:"current_version"`
	Status         string             `json:"status"`
	TableName      string             `json:"table_name"`
	TableColumns   []schema.TableData `json:"table_columns"`
}

// New accepts the metadata database pool used for queries and transactions.
func New(db contractDB, encKey string) *ContractApi {
	queries := sqlc.New(db)
	return &ContractApi{
		db:           queries,
		transactions: db,
		inspection:   schema.NewInspector(queries, encKey),
	}
}

func (c *ContractApi) InitContract(ctx context.Context, args ContractParams) (*InitContractModel, error) {
	orgId, err := util.ValidateOrgId(ctx)
	if err != nil {
		return nil, err
	}

	userId, err := util.ValidateUserId(ctx)
	if err != nil {
		return nil, err
	}
	//Fetch tables after validating schema
	tableSnapshot, err := c.inspection.ContractValidation(ctx, orgId, args.ConnectionId, args.SchemaName, args.TableName)
	if err != nil {
		return nil, err
	}
	return c.initContractSnapshot(ctx, orgId, userId, args, tableSnapshot)
}

func (c *ContractApi) ActivateContract(ctx context.Context, args ActivateContract) error {
	orgId, err := util.ValidateOrgId(ctx)
	if err != nil {
		return err
	}

	userId, err := util.ValidateUserId(ctx)
	if err != nil {
		return err
	}

	// Lock the draft and persist activation and any new version atomically.
	tx, err := c.transactions.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	transactional := &ContractApi{
		db:         c.db.WithTx(tx),
		inspection: c.inspection,
	}

	contract, err := transactional.db.GetContractForUpdate(ctx, sqlc.GetContractForUpdateParams{
		ID:    args.ID,
		OrgID: orgId,
	})
	if err != nil {
		return err
	}
	if contract.Status != "draft" {
		return fmt.Errorf("operation cannot be activated. contract is draft")
	}

	//Get ConnectionID to ValidateSchema on Activation
	tableSnapshot, err := transactional.inspection.ContractValidation(ctx, orgId, contract.ConnectionID, contract.SchemaName, contract.TableName)
	if err != nil {
		//Error fetching live schema
		return err
	}
	//Get Draft contract version
	currentVersion, err := transactional.db.GetContractVersionByContractId(ctx, args.ID)
	if err != nil {
		return err
	}

	versionNumber := currentVersion.VersionNumber
	//Compare Schema At the point of Activation,
	schemaComparison, err := ValidateContractAgainstSchema(currentVersion.SchemaSnapshot, tableSnapshot)
	if err != nil {
		return err
	}
	// Unchanged schemas retain their version without inserting columns.
	if len(schemaComparison) > 0 {
		versionNumber++
		contractVersion, err := transactional.ContractVersion(ctx, ContractVersion{
			ContractId: args.ID,
			Version:    versionNumber,
			User:       userId,
			Snapshot:   tableSnapshot,
		})
		if err != nil {
			return err
		}
		if err := transactional.ContractColumn(ctx, contractVersion.ID, tableSnapshot); err != nil {
			return err
		}
	}
	if err := transactional.activateContract(ctx, args.ID, orgId, versionNumber); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit contract activation: %w", err)
	}
	return nil

}

func (c *ContractApi) activateContract(ctx context.Context, contractId, orgId pgtype.UUID, version int32) error {
	rows, err := c.db.ActivateContract(ctx, sqlc.ActivateContractParams{
		ID:             contractId,
		CurrentVersion: version,
		OrgID:          orgId,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("contract is not draft or does not exist")
	}
	return nil

}

func (c *ContractApi) initContractSnapshot(ctx context.Context, orgId, userId pgtype.UUID, args ContractParams, tableSnapshot *schema.TableSnapshot) (*InitContractModel, error) {
	tx, err := c.transactions.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin contract initialization: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	// Use a separate instance so concurrent calls never share transaction state.
	transactional := &ContractApi{db: c.db.WithTx(tx)}
	//Create Contract
	contractModel, err := transactional.CreateContract(ctx, orgId, args)
	if err != nil {
		return nil, err
	}
	//Create ContractVersion
	contractVersion, err := transactional.ContractVersion(ctx, ContractVersion{
		ContractId:  contractModel.ID,
		Version:     int32(1),
		User:        userId,
		Snapshot:    tableSnapshot,
		Description: args.Description,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit contract initialization: %w", err)
	}

	return &InitContractModel{
		ContractID:     contractModel.ID,
		CurrentVersion: contractVersion.Version,
		Status:         contractModel.Status,
		TableName:      args.TableName,
		TableColumns:   tableSnapshot.Columns,
	}, nil
}

func (c *ContractApi) CreateContract(ctx context.Context, orgId pgtype.UUID, args ContractParams) (*ContractModel, error) {
	if _, err := c.db.GetTeam(ctx, sqlc.GetTeamParams{ID: args.TeamId, OrgID: orgId}); err != nil {
		return nil, fmt.Errorf("validate producer team: %w", err)
	}
	var freshness pgtype.Int4
	if args.FreshnessSlaHours != nil {
		freshness = pgtype.Int4{Int32: *args.FreshnessSlaHours, Valid: true}
	}
	contract, err := c.db.CreateContract(ctx, sqlc.CreateContractParams{
		OrgID:             orgId,
		ConnectionID:      args.ConnectionId,
		DisplayName:       args.DisplayName,
		ProducerTeamID:    args.TeamId,
		SchemaName:        args.SchemaName,
		TableName:         args.TableName,
		Description:       args.Description,
		Status:            "draft",
		FreshnessSlaHours: freshness,
	})
	if err != nil {
		return nil, err
	}
	return &ContractModel{
		ID:                contract.ID,
		ConnectionID:      contract.ConnectionID,
		ProducerTeamID:    contract.ProducerTeamID,
		DisplayName:       contract.DisplayName,
		SchemaName:        contract.SchemaName,
		TableName:         contract.TableName,
		Description:       contract.Description,
		Status:            contract.Status,
		FreshnessSlaHours: contract.FreshnessSlaHours,
	}, nil
}

func (c *ContractApi) ContractVersion(ctx context.Context, args ContractVersion) (*ContractVersionModel, error) {
	data, err := json.Marshal(args.Snapshot)
	if err != nil {
		return nil, err
	}
	contractVersion, err := c.db.CreateContractVersion(ctx, sqlc.CreateContractVersionParams{
		ContractID:     args.ContractId,
		VersionNumber:  args.Version,
		SchemaSnapshot: data,
		ChangeSummary:  args.Description,
		CreatedBy:      args.User,
	})

	if err != nil {
		return nil, err
	}
	return &ContractVersionModel{
		ID:          contractVersion.ID,
		ContractId:  contractVersion.ContractID,
		Version:     contractVersion.VersionNumber,
		User:        contractVersion.CreatedBy,
		Description: contractVersion.ChangeSummary,
	}, nil
}

func (c *ContractApi) ContractColumn(ctx context.Context, versionId pgtype.UUID, args *schema.TableSnapshot) error {
	var columns []sqlc.InsertContractColumnsParams
	for _, tableData := range args.Columns {
		columns = append(columns, sqlc.InsertContractColumnsParams{
			ContractVersionID: versionId,
			ColumnName:        tableData.ColumnName,
			DataType:          tableData.DataType,
			IsNullable:        tableData.IsNullable,
			IsPrimaryKey:      tableData.IsPrimaryKey,
			OrdinalPosition:   tableData.OrdinalPosition,
		})

	}

	_, err := c.db.InsertContractColumns(ctx, columns)
	if err != nil {
		return err
	}
	return nil

}
