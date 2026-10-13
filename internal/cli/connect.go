package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgtype"
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
	bridge *Bridge
}

func NewConnectCmd(bridge *Bridge) ConnectCmd {
	return ConnectCmd{bridge: bridge}
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
	output := "table"
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved database connections",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "table" && output != "json" {
				return fmt.Errorf("output must be table or json")
			}
			var storedConnections []schema.StoredConnection
			err := connectCmd.bridge.Do(cmd.Context(), http.MethodGet, "/connections", nil, &storedConnections)
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
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No saved connections.")
				return nil
			}
			for _, connection := range storedConnections {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s:%d/%s\t%s\n",
					connection.ID.String(), connection.DisplayName, connection.Host,
					connection.Port, connection.DatabaseName, connection.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&output, "output", "table", "Output format: table or json")
	return cmd
}

// connectionResult matches the API's response for add and test operations.
type connectionResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func writeConnectionResult(cmd *cobra.Command, output, text string, result connectionResult) error {
	if result.Status != "ok" {
		return fmt.Errorf("unexpected connection response status %q: %s", result.Status, result.Message)
	}
	if output == "json" {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

func (connectCmd *ConnectCmd) TestConnectionCmd() *cobra.Command {
	output := "text"
	cmd := &cobra.Command{
		Use: "test <connection-id>", Short: "Test a saved database connection", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			var id pgtype.UUID
			if err := id.Scan(args[0]); err != nil || !id.Valid {
				return fmt.Errorf("invalid connection UUID")
			}
			var result connectionResult
			if err := connectCmd.bridge.Do(cmd.Context(), http.MethodPost, "/connections/"+id.String()+"/test", nil, &result); err != nil {
				return err
			}
			return writeConnectionResult(cmd, output, "success", result)
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	return cmd
}

func (connectCmd *ConnectCmd) AddConnectionCmd() *cobra.Command {
	output := "text"
	cmd := &cobra.Command{
		Use: "add", Short: "Add a new external database to ratify", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			connDetails, err := promptConnectionDetails(cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			port, err := strconv.Atoi(connDetails.Port)
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
			var result connectionResult
			err = connectCmd.bridge.Do(cmd.Context(), http.MethodPost, "/connections", schema.ConnectionParams{
				Host: connDetails.Host, Port: port, Username: connDetails.Username, Password: connDetails.Password,
				DatabaseName: connDetails.DatabaseName, SSLMode: connDetails.SSLMode,
				SSlEnabled: connDetails.SSLMode != string(schema.DISABLED), DriverName: "pgx",
			}, &result)
			if err != nil {
				return err
			}
			return writeConnectionResult(cmd, output, "Connection added successfully.", result)
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	return cmd
}

func promptConnectionDetails(output io.Writer) (*ConnectionDetails, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Database connection parameters")
	hostname, err := prompt(reader, output, "Hostname")
	if err != nil {
		return nil, err
	}
	port, err := prompt(reader, output, "Port [5432]")
	if err != nil {
		return nil, err
	}
	if port == "" {
		port = "5432"
	}

	databaseName, err := prompt(reader, output, "Database")
	if err != nil {
		return nil, err
	}
	username, err := prompt(reader, output, "Username")
	if err != nil {
		return nil, err
	}
	sslMode, err := prompt(reader, output, "SSL mode [disable]")
	if err != nil {
		return nil, err
	}
	if sslMode == "" {
		sslMode = string(schema.DISABLED)
	}

	fmt.Println(output, "Password: ")
	passBytes, err := term.ReadPassword(syscall.Stdin)
	fmt.Println(output)
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

func prompt(reader *bufio.Reader, output io.Writer, label string) (string, error) {
	fmt.Println(output, "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}
