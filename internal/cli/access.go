package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ratifydata/ratify/internal/access"
	"github.com/spf13/cobra"
)

type AccessCmd struct {
	bridge *Bridge
}

func NewAccessCmd(bridge *Bridge) AccessCmd {
	return AccessCmd{bridge: bridge}
}

func (c *AccessCmd) TeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage team access",
		Long:  "Manage team access",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	cmd.AddGroup(&cobra.Group{
		ID:    "team",
		Title: "Team Access",
	})
	cmd.AddCommand(c.AddTeam())
	cmd.AddCommand(c.ListTeam())
	return cmd

}

func (c *AccessCmd) AddTeam() *cobra.Command {
	var params access.TeamParams
	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Create a team",
		Example: `  ratify team add --name "Data Platform" --email data@example.com --description "Owns data pipelines"`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			params.Name = strings.TrimSpace(params.Name)
			if params.Name == "" {
				return fmt.Errorf("team name is required")
			}
			params.EmailAddress = strings.TrimSpace(params.EmailAddress)
			if params.EmailAddress == "" {
				return fmt.Errorf("email address is required")
			}
			ctx := cmd.Context()
			var response struct {
				Body access.OrgTeam `json:"body"`
			}
			err := c.bridge.Do(ctx, http.MethodPost, "/teams", params, &response)
			if err != nil {
				return fmt.Errorf("create team: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created team %s (%s)\n", response.Body.Name, response.Body.ID.String())
			return err
		},
	}
	cmd.Flags().StringVar(&params.Name, "name", "", "Team name (required)")
	cmd.Flags().StringVar(&params.EmailAddress, "email", "", "Team email address (required)")
	cmd.Flags().StringVar(&params.Description, "description", "", "Team description")
	return cmd
}

func (c *AccessCmd) ListTeam() *cobra.Command {
	output := "table"
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List teams",
		Example: "ratify team list",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var teams []access.OrgTeam
			err := c.bridge.Do(cmd.Context(), http.MethodGet, "/teams", nil, &teams)
			if err != nil {
				return err
			}
			if output == "json" {
				if teams == nil {
					teams = make([]access.OrgTeam, 0)
				}
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(teams)
			}

			for _, team := range teams {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n\n",
					team.Name, team.Description, team.EmailAddress)
			}
			return nil

		},
	}
	cmd.Flags().StringVar(&output, "output", "table", "Output format (json|table)")
	return cmd

}
