package cmd

import "github.com/spf13/cobra"

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage custom agent roles",
}

func init() {
	rootCmd.AddCommand(roleCmd)
}
