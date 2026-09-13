package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ratify",
	Short: "A data contract workflow engine",
	Long:  `A command line tool for data contract workflow engine`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to ratify, A data contract workflow engine")
	},
}
