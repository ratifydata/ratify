package contract

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/ratifydata/ratify/internal/util"
)

type ContractApi struct {
	db         *sqlc.Queries
	inspection *schema.Inspector
}

type ContractParams struct {
	ConnectionId      pgtype.UUID `json:"connection_id"`
	TeamId            pgtype.UUID `json:"team_id"`
	DisplayName       string      `json:"display_name"`
	Description       pgtype.Text `json:"description"`
	FreshnessSlaHours int32       `json:"freshness_sla_hours"`
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
	Contstraints      []byte      `json:"contstraints"`
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

type ContractVersionModel struct {
	ID          pgtype.UUID `json:"contract_version_id"`
	ContractId  pgtype.UUID `json:"contract_id"`
	Version     int32       `json:"version"`
	User        pgtype.UUID `json:"user"`
	Description pgtype.Text `json:"description"`
}

type InitContractModel struct {
	ContactId      pgtype.UUID        `json:"contact_id"`
	CurrentVersion int32              `json:"current_version"`
	Status         string             `json:"status"`
	TableName      string             `json:"table_name"`
	TableColumns   []schema.TableData `json:"table_columns"`
}

func New(db *sqlc.Queries, encKey string) *ContractApi {
	return &ContractApi{
		db:         db,
		inspection: schema.NewInspector(db, encKey),
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
	tableSnapshot, err := c.inspection.ContractValidation(ctx, args.ConnectionId, args.TableName)
	if err != nil {
		return nil, err
	}
	//Create Contract
	contractModel, err := c.CreateContract(ctx, orgId, args)
	if err != nil {
		return nil, err
	}
	//Create ContractVersion
	contractVersion, err := c.ContractVersion(ctx, ContractVersion{
		ContractId:  contractModel.ID,
		Version:     int32(1),
		User:        userId,
		Snapshot:    tableSnapshot,
		Description: args.Description,
	})
	if err != nil {
		return nil, err
	}
	if err = c.ContractColumn(ctx, contractVersion.ID, tableSnapshot); err != nil {
		return nil, err
	}

	return &InitContractModel{
		ContactId:      contractModel.ID,
		CurrentVersion: contractVersion.Version,
		Status:         contractModel.Status,
		TableName:      args.TableName,
		TableColumns:   tableSnapshot.Columns,
	}, nil
}

func (c *ContractApi) CreateContract(ctx context.Context, orgId pgtype.UUID, args ContractParams) (*ContractModel, error) {
	contract, err := c.db.CreateContract(ctx, sqlc.CreateContractParams{
		OrgID:          orgId,
		ConnectionID:   args.ConnectionId,
		DisplayName:    args.DisplayName,
		ProducerTeamID: args.TeamId,
		SchemaName:     args.DisplayName,
		TableName:      args.DisplayName,
		Description:    args.Description,
		FreshnessSlaHours: pgtype.Int4{
			Int32: args.FreshnessSlaHours,
			Valid: true,
		},
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
		ID:         contractVersion.ID,
		ContractId: contractVersion.ContractID,
		Version:    contractVersion.VersionNumber,
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
			//Todo: Update the key to something fetched from the Schema
			IsPrimaryKey:    false,
			OrdinalPosition: tableData.OrdinalPosition,
		})

	}

	_, err := c.db.InsertContractColumns(ctx, columns)
	if err != nil {
		return err
	}
	return nil

}
