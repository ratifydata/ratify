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
