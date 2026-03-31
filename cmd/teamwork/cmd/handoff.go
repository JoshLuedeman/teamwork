package cmd

import "github.com/spf13/cobra"

var handoffCmd = &cobra.Command{
	Use:   "handoff",
	Short: "Manage handoff artifacts for workflow transitions",
}

func init() {
	rootCmd.AddCommand(handoffCmd)
}
