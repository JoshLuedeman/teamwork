package cmd

import (
	"fmt"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/installer"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Teamwork framework files to the latest version",
	Long: `Update fetches the latest framework files from the upstream Teamwork repository
and applies changes. Files that have been modified locally are skipped with a
warning unless --force is set.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().String("source", "JoshLuedeman/teamwork", "Source repository (owner/repo)")
	updateCmd.Flags().String("ref", "main", "Git ref to update to (branch, tag, or SHA)")
	updateCmd.Flags().Bool("force", false, "Overwrite user-modified files without warning")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		return err
	}
	ref, err := cmd.Flags().GetString("ref")
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	owner, repo, err := parseUpdateSource(source)
	if err != nil {
		return err
	}

	return installer.Update(dir, owner, repo, ref, force)
}

func parseUpdateSource(source string) (string, string, error) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --source format %q: expected owner/repo", source)
	}
	return parts[0], parts[1], nil
}
