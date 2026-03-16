package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/spf13/cobra"
)

var mcpConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate MCP client configuration",
	Long:  "Generate paste-ready JSON configuration for MCP clients such as Claude Desktop or VS Code.",
	RunE:  runMCPConfig,
}

func init() {
	mcpConfigCmd.Flags().String("format", "claude-desktop", "Output format: claude-desktop or vscode")
	mcpConfigCmd.Flags().Bool("only-ready", false, "Exclude servers with missing environment variables")
	mcpCmd.AddCommand(mcpConfigCmd)
}

func runMCPConfig(cmd *cobra.Command, _ []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return &ExitError{Code: 1, Message: fmt.Sprintf("failed to load config: %v", err)}
	}

	format, _ := cmd.Flags().GetString("format")
	onlyReady, _ := cmd.Flags().GetBool("only-ready")

	// Build server entries in sorted order for deterministic output.
	names := make([]string, 0, len(cfg.MCPServers))
	for name := range cfg.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)

	servers := make(map[string]any)
	for _, name := range names {
		srv := cfg.MCPServers[name]

		if onlyReady && !isServerReady(srv) {
			continue
		}

		entry := buildServerEntry(srv)
		servers[name] = entry
	}

	var wrapper map[string]any
	switch format {
	case "vscode":
		wrapper = map[string]any{"servers": servers}
	default: // claude-desktop
		wrapper = map[string]any{"mcpServers": servers}
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(wrapper)
}

// isServerReady returns true if all required env vars are set.
func isServerReady(srv config.MCPServer) bool {
	for _, v := range srv.EnvVars {
		if os.Getenv(v) == "" {
			return false
		}
	}
	return true
}

// buildServerEntry creates the JSON-ready map for an MCP server entry.
func buildServerEntry(srv config.MCPServer) map[string]any {
	entry := make(map[string]any)

	if srv.URL != "" {
		entry["type"] = "http"
		entry["url"] = srv.URL
	} else if srv.Command != "" {
		parts := strings.Fields(srv.Command)
		entry["type"] = "stdio"
		entry["command"] = parts[0]
		if len(parts) > 1 {
			entry["args"] = parts[1:]
		}
	}

	if len(srv.EnvVars) > 0 {
		envMap := make(map[string]string)
		for _, v := range srv.EnvVars {
			envMap[v] = "${" + v + "}"
		}
		entry["env"] = envMap
	}

	return entry
}
