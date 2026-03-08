package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/config"
	"github.com/spf13/cobra"
)

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured MCP servers",
	RunE:  runMCPList,
}

func init() {
	mcpListCmd.Flags().String("role", "", "Filter servers by role")
	mcpListCmd.Flags().Bool("json", false, "Output as JSON array")
	mcpCmd.AddCommand(mcpListCmd)
}

// mcpServerStatus returns a status string for an MCP server based on its env vars.
func mcpServerStatus(envVars []string) string {
	if len(envVars) == 0 {
		return "✓ ready"
	}
	for _, v := range envVars {
		if os.Getenv(v) == "" {
			return "✗ missing"
		}
	}
	return "✓ set"
}

// mcpServerJSON is the JSON representation of an MCP server in list output.
type mcpServerJSON struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url,omitempty"`
	Command     string   `json:"command,omitempty"`
	Roles       []string `json:"roles"`
	EnvVars     []string `json:"env_vars"`
	Status      string   `json:"status"`
}

func runMCPList(cmd *cobra.Command, _ []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return &ExitError{Code: 1, Message: fmt.Sprintf("failed to load config: %v", err)}
	}

	if len(cfg.MCPServers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured. See docs/mcp.md for setup instructions.")
		return nil
	}

	roleFilter, _ := cmd.Flags().GetString("role")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Collect and sort server names for deterministic output.
	names := make([]string, 0, len(cfg.MCPServers))
	for name := range cfg.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)

	// Apply role filter.
	var filtered []string
	for _, name := range names {
		srv := cfg.MCPServers[name]
		if roleFilter != "" && !containsRole(srv.Roles, roleFilter) {
			continue
		}
		filtered = append(filtered, name)
	}

	if jsonOutput {
		return mcpListJSON(cmd, cfg, filtered)
	}
	return mcpListTable(cmd, cfg, filtered)
}

func mcpListTable(cmd *cobra.Command, cfg *config.Config, names []string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%-12s  %-30s  %-18s  %s\n", "SERVER", "ROLES", "ENV VARS", "STATUS")
	for _, name := range names {
		srv := cfg.MCPServers[name]
		roles := strings.Join(srv.Roles, ", ")
		envVars := strings.Join(srv.EnvVars, ", ")
		if envVars == "" {
			envVars = "(none)"
		}
		status := mcpServerStatus(srv.EnvVars)
		fmt.Fprintf(out, "%-12s  %-30s  %-18s  %s\n", name, roles, envVars, status)
	}
	return nil
}

func mcpListJSON(cmd *cobra.Command, cfg *config.Config, names []string) error {
	servers := make([]mcpServerJSON, 0, len(names))
	for _, name := range names {
		srv := cfg.MCPServers[name]
		servers = append(servers, mcpServerJSON{
			Name:        name,
			Description: srv.Description,
			URL:         srv.URL,
			Command:     srv.Command,
			Roles:       srv.Roles,
			EnvVars:     srv.EnvVars,
			Status:      mcpServerStatus(srv.EnvVars),
		})
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(servers)
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
