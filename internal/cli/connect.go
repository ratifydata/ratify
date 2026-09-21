package cli

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type ConnectionDetails struct {
	DatabaseName string
	Username     string
	Password     string
	Host         string
	Port         string
	SSLMode      string
}

type connectionInspector interface {
	ListDatabaseConnections(context.Context) ([]schema.StoredConnection, error)
	TestConnection(context.Context, pgtype.UUID) error
	SchemaInspection(context.Context, schema.ConnectionParams) error
}

type ConnectCmd struct {
	inspector     connectionInspector
	promptDetails func() (*ConnectionDetails, error)
}

var orgID string

func NewConnectCmd(db *sqlc.Queries, encKey string) ConnectCmd {
	return ConnectCmd{
		inspector: schema.NewInspector(db, encKey),
	}
}

func (connectCmd *ConnectCmd) Connect() *cobra.Command {
	var mainCommand = &cobra.Command{
		Use:   "connect",
		Short: "Links to user's external database,list and verify connectivity",
		Long:  "Links to user's external database,list and verify connectivity",
		Run: func(cmd *cobra.Command, args []string) {

		}}
	mainCommand.AddGroup(&cobra.Group{
		ID:    "connection",
		Title: "Connection",
	})
	mainCommand.AddCommand(connectCmd.TestConnectionCmd())
	mainCommand.AddCommand(connectCmd.ListConnections())
	mainCommand.AddCommand(connectCmd.AddConnectionCmd())
	mainCommand.PersistentFlags().StringVar(&orgID, "org-id", "", "Ratify organization UUID")
	_ = mainCommand.MarkPersistentFlagRequired("org-id")
	return mainCommand

}

func (connectCmd *ConnectCmd) ListConnections() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved database connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := connectionContext(cmd)
			if err != nil {
				return err
			}
			storedConnections, err := connectCmd.inspector.ListDatabaseConnections(ctx)
			if err != nil {
				return err
			}
			if len(storedConnections) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No saved connections.")
				return nil
			}
			for _, connection := range storedConnections {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s:%d/%s\t%s\n",
					connection.ID.String(), connection.DisplayName, connection.Host,
					connection.Port, connection.DatabaseName, connection.Status)
			}
			return nil
		},
	}
}

func (connectCmd *ConnectCmd) TestConnectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <connection-id>",
		Short: "Test a saved database connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var connectionID pgtype.UUID
			if err := connectionID.Scan(args[0]); err != nil {
				return fmt.Errorf("invalid connection ID: %w", err)
			}

			err := connectCmd.inspector.TestConnection(cmd.Context(), connectionID)

			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Connection test passed.")
			return nil
		},
	}
}

func (connectCmd *ConnectCmd) AddConnectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new external database to ratify",
		Long:  "Add a new external database to ratify",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := connectionContext(cmd)
			if err != nil {
				return err
			}
			promptDetails := connectCmd.promptDetails
			if promptDetails == nil {
				promptDetails = promptConnectionDetails
			}
			connDetails, err := promptDetails()
			if err != nil {
				return err
			}

			port, err := strconv.Atoi(connDetails.Port)
			if err != nil {
				slog.Error("Error reading port number")
				return err
			}

			err = connectCmd.inspector.SchemaInspection(ctx, schema.ConnectionParams{
				Host:         connDetails.Host,
				Port:         port,
				Username:     connDetails.Username,
				Password:     connDetails.Password,
				DatabaseName: connDetails.DatabaseName,
				SSlEnabled:   connDetails.SSLMode != string(schema.DISABLED),
				DriverName:   "pgx",
			})
			if err != nil {
				slog.Error("Error connecting to external database from CLI")
				return err
			}

			return nil
		},
	}
}

// connectionContext supplies the organization expected by the schema inspector.
func connectionContext(cmd *cobra.Command) (context.Context, error) {
	value, err := cmd.Flags().GetString("org-id")
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, fmt.Errorf("organization ID is required")
	}
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}
	return context.WithValue(cmd.Context(), "OrgID", id), nil
}

func promptConnectionDetails() (*ConnectionDetails, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Database connection parameters. \n")
	hostname, err := prompt(reader, "Hostname")
	if err != nil {
		return nil, err
	}
	port, err := prompt(reader, "Port [5432]")
	if err != nil {
		return nil, err
	}
	if port == "" {
		port = "5432"
	}

	databaseName, err := prompt(reader, "Database")
	if err != nil {
		return nil, err
	}
	username, err := prompt(reader, "Username")
	if err != nil {
		return nil, err
	}
	sslMode, err := prompt(reader, "SSL mode [disable]")
	if err != nil {
		return nil, err
	}
	if sslMode == "" {
		sslMode = string(schema.DISABLED)
	}

	fmt.Print("Password: ")
	passBytes, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	fmt.Println("Confirm Password: ")
	confirmPass, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	if string(passBytes) != string(confirmPass) {
		return nil, fmt.Errorf("password mismatch")
	}

	password := string(passBytes)
	return &ConnectionDetails{
		DatabaseName: databaseName,
		Host:         hostname,
		Username:     username,
		Password:     password,
		Port:         port,
		SSLMode:      sslMode,
	}, nil
}

func prompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
