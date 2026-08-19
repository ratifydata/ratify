package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "ratify",
	Short: "A data contract workflow engine",
	Long:  `A command line tool for data contract workflow engine`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}
