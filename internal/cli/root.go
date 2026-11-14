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
	"context"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	if err := newRootCmd().ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	bridge := NewBridge()
	connectionCmd := NewConnectCmd(bridge)
	accessCmd := NewAccessCmd(bridge)
	rootCmd := &cobra.Command{
		Use:          "ratify",
		Short:        "A data contract workflow engine",
		Long:         `A command line tool for data contract workflow engine`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd == cmd.Root() || cmd == authCmd {
				return nil
			}
			return bridge.Configure(cmd)
		},
	}
	rootCmd.PersistentFlags().String("api-url", "", "Ratify server URL (overrides RATIFY_API_URL and saved API_URL; default http://localhost:8080)")
	rootCmd.AddCommand(connectionCmd.Connect(), authCmd, accessCmd.TeamCmd())
	return rootCmd
}
