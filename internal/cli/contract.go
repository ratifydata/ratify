// Copyright (C) 2026 Ratify Data
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/access"
	"github.com/ratifydata/ratify/internal/contract"
	"github.com/ratifydata/ratify/internal/schema"
	"github.com/spf13/cobra"
)

type ContractCmd struct{ bridge *Bridge }

func NewContractCmd(bridge *Bridge) ContractCmd { return ContractCmd{bridge: bridge} }

func (c *ContractCmd) Contract() *cobra.Command {
	cmd := &cobra.Command{Use: "contract", Short: "Create, show, list, activate, and deprecate data contracts", Args: cobra.NoArgs}
	cmd.AddCommand(c.Create(), c.List(), c.Activate(), c.Display(), c.Deprecate())
	return cmd
}

type contractResponse[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Body    T      `json:"body"`
}

func (r contractResponse[T]) check() error {
	if r.Status != "ok" {
		if r.Message != "" {
			return fmt.Errorf("contract request failed: %s", r.Message)
		}
		return fmt.Errorf("unexpected contract response status %q", r.Status)
	}
	return nil
}

func contractUUID(value, label string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil || !id.Valid {
		return id, fmt.Errorf("invalid %s UUID", label)
	}
	return id, nil
}

func writeContractJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (c *ContractCmd) Create() *cobra.Command {
	var connectionID, teamID, name, description, schemaName, tableName, output string
	var freshness int32
	var interactive bool
	cmd := &cobra.Command{
		Use: "create", Short: "Create a draft contract from a database table", Args: cobra.NoArgs,
		Example: `  ratify contract create --connection-id <uuid> --team-id <uuid> --name "Orders" --schema public --table orders`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			// Reject invalid supplied flags before fetching lists or asking for input.
			for _, field := range []struct{ value, label string }{{connectionID, "connection"}, {teamID, "team"}} {
				if field.value != "" {
					if _, err := contractUUID(field.value, field.label); err != nil {
						return err
					}
				}
			}
			for _, field := range []struct{ flag, value string }{{"connection-id", connectionID}, {"team-id", teamID}, {"name", name}, {"schema", schemaName}, {"table", tableName}} {
				if cmd.Flags().Changed(field.flag) && strings.TrimSpace(field.value) == "" {
					return fmt.Errorf("%s must not be empty", field.flag)
				}
			}
			if cmd.Flags().Changed("freshness-sla-hours") && freshness < 0 {
				return fmt.Errorf("freshness-sla-hours must be nonnegative")
			}
			if interactive || connectionID == "" || teamID == "" || name == "" || tableName == "" {
				if err := c.promptCreate(cmd, &connectionID, &teamID, &schemaName, &tableName, &name, &description, &freshness); err != nil {
					return err
				}
			}
			connection, err := contractUUID(connectionID, "connection")
			if err != nil {
				return err
			}
			team, err := contractUUID(teamID, "team")
			if err != nil {
				return err
			}
			name, schemaName, tableName = strings.TrimSpace(name), strings.TrimSpace(schemaName), strings.TrimSpace(tableName)
			if name == "" || schemaName == "" || tableName == "" {
				return fmt.Errorf("name, schema, and table are required")
			}
			params := contract.ContractParams{ConnectionId: connection, TeamId: team, DisplayName: name, SchemaName: schemaName, TableName: tableName,
				Description: pgtype.Text{String: description, Valid: cmd.Flags().Changed("description")}}
			if cmd.Flags().Changed("freshness-sla-hours") {
				if freshness < 0 {
					return fmt.Errorf("freshness-sla-hours must be nonnegative")
				}
				params.FreshnessSlaHours = &freshness
			}
			if c.bridge == nil {
				return fmt.Errorf("API connection is required")
			}
			var response contractResponse[contract.InitContractModel]
			if err := c.bridge.Do(cmd.Context(), http.MethodPost, "/contracts", params, &response); err != nil {
				return fmt.Errorf("create contract: %w", err)
			}
			if err := response.check(); err != nil {
				return err
			}
			if !response.Body.ContractID.Valid || response.Body.CurrentVersion < 1 || response.Body.Status == "" {
				return fmt.Errorf("create contract response has no valid contract details")
			}
			if output == "json" {
				return writeContractJSON(cmd, response.Body)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created contract %s (version %d, %s)\n", response.Body.ContractID.String(), response.Body.CurrentVersion, response.Body.Status)
			return err
		},
	}
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Select a connection and team, then enter contract details")
	cmd.Flags().StringVar(&connectionID, "connection-id", "", "Database connection UUID (required)")
	cmd.Flags().StringVar(&teamID, "team-id", "", "Producer team UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Contract display name (required)")
	cmd.Flags().StringVar(&schemaName, "schema", "public", "Database schema")
	cmd.Flags().StringVar(&tableName, "table", "", "Database table (required)")
	cmd.Flags().StringVar(&description, "description", "", "Contract description")
	cmd.Flags().Int32Var(&freshness, "freshness-sla-hours", 0, "Freshness SLA in hours (optional)")
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	return cmd
}

func (c *ContractCmd) Display() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use: "show [id]", Short: "Show contract details, columns, producer, consumers, and version history", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			id, err := contractUUID(args[0], "contract")
			if err != nil {
				return err
			}
			var response contractResponse[*contract.ContractModel]
			if err := c.bridge.Do(cmd.Context(), http.MethodGet, "/contracts/"+id.String(), nil, &response); err != nil {
				return fmt.Errorf("show contract: %w", err)
			}
			if err := response.check(); err != nil {
				return err
			}
			if response.Body == nil {
				return fmt.Errorf("contract response has no details")
			}
			if output == "json" {
				return writeContractJSON(cmd, response.Body)
			}
			return displayContract(cmd, response.Body)
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	return cmd
}

func (c *ContractCmd) List() *cobra.Command {
	var output string
	cmd := &cobra.Command{Use: "list", Short: "List contracts, optionally filtered by status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "table" && output != "json" {
				return fmt.Errorf("output must be table or json")
			}
			path := "/contracts"

			var response contractResponse[[]contract.ContractModel]
			if err := c.bridge.Do(cmd.Context(), http.MethodGet, path, nil, &response); err != nil {
				return fmt.Errorf("list contracts: %w", err)
			}
			if err := response.check(); err != nil {
				return err
			}
			if response.Body == nil {
				response.Body = []contract.ContractModel{}
			}
			if output == "json" {
				return writeContractJSON(cmd, response.Body)
			}
			if len(response.Body) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No contracts.")
				return err
			}
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(writer, "ID\tNAME\tTABLE\tSTATUS\tVERSION"); err != nil {
				return err
			}
			for _, item := range response.Body {
				if _, err := fmt.Fprintf(writer, "%s\t%s\t%s.%s\t%s\t%d\n", item.ID.String(), item.DisplayName, item.SchemaName, item.TableName, item.Status, item.CurrentVersion); err != nil {
					return err
				}
			}
			return writer.Flush()
		},
	}
	cmd.Flags().StringVar(&output, "output", "table", "Output format: table or json")
	return cmd
}

func (c *ContractCmd) Activate() *cobra.Command {
	var output string
	cmd := &cobra.Command{Use: "activate <contract-id>", Short: "Activate a draft contract", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("output must be text or json")
			}
			id, err := contractUUID(args[0], "contract")
			if err != nil {
				return err
			}
			var response contractResponse[any]
			if err := c.bridge.Do(cmd.Context(), http.MethodPost, "/contracts/"+id.String()+"/activate", contract.ActivateContract{ID: id}, &response); err != nil {
				return fmt.Errorf("activate contract: %w", err)
			}
			if err := response.check(); err != nil {
				return err
			}
			if output == "json" {
				return writeContractJSON(cmd, struct {
					Status string `json:"status"`
				}{Status: response.Status})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Activated contract %s\n", id.String())
			return err
		},
	}
	cmd.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	return cmd
}

func (c *ContractCmd) Deprecate() *cobra.Command {
	return &cobra.Command{
		Use: "deprecate [id]", Short: "Deprecate an active contract after confirmation", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := contractUUID(args[0], "contract")
			if err != nil {
				return err
			}
			if c.bridge == nil {
				return fmt.Errorf("API connection is required")
			}
			reader := bufio.NewReader(cmd.InOrStdin())
			for {
				answer, err := readContractInput(reader, cmd.ErrOrStderr(), fmt.Sprintf("Deprecate contract %s? [y/n]", id.String()))
				if err != nil {
					return err
				}
				switch strings.ToLower(answer) {
				case "n":
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Deprecation canceled.")
					return err
				case "y":
					var response contractResponse[any]
					if err := c.bridge.Do(cmd.Context(), http.MethodPatch, "/contracts/"+id.String()+"/deprecate", nil, &response); err != nil {
						return fmt.Errorf("deprecate contract: %w", err)
					}
					if err := response.check(); err != nil {
						return err
					}
					_, err := fmt.Fprintf(cmd.OutOrStdout(), "Deprecated contract %s\n", id.String())
					return err
				default:
					if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "Enter y or n."); err != nil {
						return err
					}
				}
			}
		},
	}
}

// readContractInput reads one answer from the same terminal session.
func readContractInput(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	if _, err := fmt.Fprintf(out, "%s: ", label); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && !(err == io.EOF && len(value) > 0) {
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	return strings.TrimSpace(value), nil
}

func selectContractItem(reader *bufio.Reader, out io.Writer, label string, items []string) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("no %s available", label)
	}
	if _, err := fmt.Fprintf(out, "Select %s:\n", label); err != nil {
		return 0, err
	}
	for i, item := range items {
		if _, err := fmt.Fprintf(out, "%d) %s\n", i+1, item); err != nil {
			return 0, err
		}
	}
	for {
		value, err := readContractInput(reader, out, "Choice")
		if err != nil {
			return 0, err
		}
		choice, err := strconv.Atoi(value)
		if err == nil && choice >= 1 && choice <= len(items) {
			return choice - 1, nil
		}
		if _, err := fmt.Fprintf(out, "Enter a number from 1 to %d.\n", len(items)); err != nil {
			return 0, err
		}
	}
}

func (c *ContractCmd) promptCreate(cmd *cobra.Command, connection, team, schemaName, table, name, description *string, freshness *int32) error {
	if c.bridge == nil {
		return fmt.Errorf("API connection is required")
	}
	reader, out := bufio.NewReader(cmd.InOrStdin()), cmd.ErrOrStderr()
	if *connection == "" {
		var response contractResponse[[]schema.StoredConnection]
		if err := c.bridge.Do(cmd.Context(), http.MethodGet, "/connections", nil, &response); err != nil {
			return err
		}
		if err := response.check(); err != nil {
			return err
		}
		labels := make([]string, len(response.Body))
		for i, item := range response.Body {
			labels[i] = fmt.Sprintf("%s (%s, %s)", item.DisplayName, item.DatabaseName, item.ID.String())
		}
		choice, err := selectContractItem(reader, out, "connection", labels)
		if err != nil {
			return err
		}
		*connection = response.Body[choice].ID.String()
	}
	if _, err := contractUUID(*connection, "connection"); err != nil {
		return err
	}
	if *team == "" {
		var response contractResponse[[]access.OrgTeam]
		if err := c.bridge.Do(cmd.Context(), http.MethodGet, "/teams", nil, &response); err != nil {
			return err
		}
		if err := response.check(); err != nil {
			return err
		}
		labels := make([]string, len(response.Body))
		for i, item := range response.Body {
			labels[i] = fmt.Sprintf("%s (%s)", item.Name, item.ID.String())
		}
		choice, err := selectContractItem(reader, out, "producer team", labels)
		if err != nil {
			return err
		}
		*team = response.Body[choice].ID.String()
	}
	if _, err := contractUUID(*team, "team"); err != nil {
		return err
	}
	// These fields are still needed by the create API; supplied flags skip prompts.
	for _, field := range []struct {
		flag, label string
		value       *string
		required    bool
	}{
		{"schema", "Schema [public]", schemaName, false},
		{"table", "Table", table, true},
		{"name", "Display name", name, true},
		{"description", "Description (optional)", description, false},
	} {
		if cmd.Flags().Changed(field.flag) {
			continue
		}
		for {
			value, err := readContractInput(reader, out, field.label)
			if err != nil {
				return err
			}
			if value == "" && field.required {
				continue
			}
			if value != "" || field.flag != "schema" {
				*field.value = value
			}
			if field.flag == "description" && value != "" {
				if err := cmd.Flags().Set("description", value); err != nil {
					return err
				}
			}
			break
		}
	}
	if !cmd.Flags().Changed("freshness-sla-hours") {
		for {
			value, err := readContractInput(reader, out, "Freshness SLA in hours (blank to omit)")
			if err != nil {
				return err
			}
			if value == "" {
				break
			}
			hours, err := strconv.ParseInt(value, 10, 32)
			if err != nil || hours < 0 {
				if _, err := fmt.Fprintln(out, "Enter a nonnegative whole number of hours."); err != nil {
					return err
				}
				continue
			}
			if err := cmd.Flags().Set("freshness-sla-hours", value); err != nil {
				return err
			}
			*freshness = int32(hours)
			break
		}
	}
	return nil
}
