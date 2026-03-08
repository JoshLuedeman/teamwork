package cmd

import "github.com/spf13/cobra"

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP server configuration",
	Long:  "List configured MCP servers and generate client configuration.",
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
