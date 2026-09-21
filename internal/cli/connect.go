package cli

import (
	"bufio"
	"encoding/json"
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

type ConnectCmd struct {
	inspector *schema.Inspector
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
	return mainCommand

}

func (connectCmd *ConnectCmd) ListConnections() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved database connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "table" && output != "json" {
				return fmt.Errorf("unsupported output format %q: use table or json", output)
			}
			storedConnections, err := connectCmd.inspector.ListDatabaseConnections(cmd.Context())
			if err != nil {
				return err
			}
			if output == "json" {
				if storedConnections == nil {
					storedConnections = []schema.StoredConnection{}
				}
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(storedConnections)
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
	cmd.Flags().StringVar(&output, "output", "table", "Output format: table or json")
	return cmd
}

func (connectCmd *ConnectCmd) TestConnectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <connection-id>",
		Short: "Test a saved database connection",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var connectionID pgtype.UUID
			if err := connectionID.Scan(args[0]); err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "failure")
				return
			}

			err := connectCmd.inspector.TestConnection(cmd.Context(), connectionID)

			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "failure")
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), "success")
			return
		},
	}
}

func (connectCmd *ConnectCmd) AddConnectionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new external database to ratify",
		Long:  "Add a new external database to ratify",
		RunE: func(cmd *cobra.Command, args []string) error {

			connDetails, err := promptConnectionDetails()
			if err != nil {
				return err
			}

			port, err := strconv.Atoi(connDetails.Port)
			if err != nil {
				slog.Error("Error reading port number")
				return err
			}

			err = connectCmd.inspector.SchemaInspection(cmd.Context(), schema.ConnectionParams{
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
		return nil, fmt.Errorf("error reading password")
	}

	fmt.Println("Confirm Password: ")
	confirmPass, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return nil, fmt.Errorf("error reading password")
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
