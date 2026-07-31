package schema

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type ConnectionParams struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
	SSLMode      string `json:"ssl_mode"`
	SSlEnable    bool   `json:"ssl_enable"`
	DriverName   string `json:"driver_name"`
}

type Inspector struct {
	db *sqlc.Queries
}

func NewInspector(db *sqlc.Queries) *Inspector {
	return &Inspector{db: db}
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
	return nil

}
