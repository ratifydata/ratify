package schema

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/auth"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type ConnectionParams struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
	SSLMode      string `json:"ssl_mode"`
	SSlEnabled   bool   `json:"ssl_enabled"`
	DriverName   string `json:"driver_name"`
}
type StoredConnection struct {
	ID           pgtype.UUID `json:"id"`
	DisplayName  string      `json:"display_name"`
	Host         string      `json:"host"`
	Port         int32       `json:"port"`
	DatabaseName string      `json:"database_name"`
	Username     string      `json:"username"`
	SSLEnabled   bool        `json:"ssl_enabled"`
	SSLMode      string      `json:"ssl_mode"`
	Status       string      `json:"status"`
}
type Inspector struct {
	db     *sqlc.Queries
	encKey string
}

func NewInspector(db *sqlc.Queries, encKey string) *Inspector {
	return &Inspector{db: db, encKey: encKey}
}

func (i *Inspector) SchemaInspection(ctx context.Context, params ConnectionParams) error {
	//Makes the Url-Escape safe
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(params.Username, params.Password),
		Host:   net.JoinHostPort(params.Host, strconv.Itoa(params.Port)),
		Path:   params.DatabaseName,
	}
	q := u.Query()
	q.Set("sslmode", params.SSLMode)
	u.RawQuery = q.Encode()

	db, err := EstablishConnection(params.DriverName, u.String())
	if err != nil {
		slog.Error("error establishing connection to driver")
		return err
	}
	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			slog.Error("error while closing connection to database")
		}
	}(db)
	//Validate Privilege can be combined with Establish Connection
	if err = ValidatePrivileges(ctx, db); err != nil {
		return err
	}

	//On connection and privilege validation store the details
	//1. We encrypt the password at rest
	//2. Save the connection
	enc, err := auth.Encrypt([]byte(i.encKey), params.Password)
	if err != nil {
		slog.Error("error encrypting password")
		return err
	}

	//Assert ctx value !nil
	orgId, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok {
		return fmt.Errorf("OrgID missing from context")
	}

	args := sqlc.CreateDatabaseConnectionParams{
		DatabaseName:      params.DatabaseName,
		Host:              params.Host,
		Port:              int32(params.Port),
		Username:          params.Username,
		PasswordEncrypted: enc.CipherText,
		Nonce:             enc.Nonce,
		OrgID:             orgId,
		SslMode:           params.SSLMode,
		SslEnabled:        params.SSlEnabled,
		Status:            "ACTIVE",
	}

	//The return entry has no use currently. Just check for errors
	_, err = i.db.CreateDatabaseConnection(ctx, args)
	if err != nil {
		slog.Error("error creating database connection")
		return err
	}
	return nil

}

func (i *Inspector) ListDatabaseConnections(ctx context.Context) ([]StoredConnection, error) {

	orgID, ok := ctx.Value("OrgID").(pgtype.UUID)
	if !ok || !orgID.Valid {
		slog.Error("OrgID missing from context")
		return nil, fmt.Errorf("OrgID missing from context")
	}

	dbConnections, err := i.db.ListDatabaseConnectionsByOrg(ctx, orgID)
	if err != nil {
		slog.Error("error listing database connections")
		return nil, err
	}
	connections := make([]StoredConnection, 0, len(dbConnections))
	for _, dbConn := range dbConnections {
		connections = append(connections, StoredConnection{
			ID:           dbConn.ID,
			DisplayName:  dbConn.DisplayName,
			Host:         dbConn.Host,
			Port:         dbConn.Port,
			DatabaseName: dbConn.DatabaseName,
			Username:     dbConn.Username,
			SSLEnabled:   dbConn.SslEnabled,
			SSLMode:      dbConn.SslMode,
			Status:       dbConn.Status,
		})
	}

	return connections, nil
}
