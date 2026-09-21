package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratifydata/ratify/internal/config"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/spf13/cobra"
)

func Execute(cfg *config.Config, pool *pgxpool.Pool) {
	ctx := context.Background()
	if err := newRootCmd(cfg, pool).ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(cfg *config.Config, pool *pgxpool.Pool) *cobra.Command {

	db := sqlc.New(pool)
	connectionCmd := NewConnectCmd(db, cfg.EncryptionKey)

	rootCmd := &cobra.Command{
		Use:   "ratify",
		Short: "A data contract workflow engine",
		Long:  `A command line tool for data contract workflow engine`,
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Welcome to ratify, A data contract workflow engine")
		},
	}
	rootCmd.AddCommand(connectionCmd.Connect())
	rootCmd.AddCommand(authCmd)
	return rootCmd
}
