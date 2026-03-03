package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/installer"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Teamwork framework files into the current project",
	Long: `Install fetches framework files from the upstream Teamwork repository and writes
them into the current project directory. Starter files (MEMORY.md, CHANGELOG.md)
are created if they do not already exist.

If the framework is already installed, use 'teamwork update' instead.`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().String("source", "JoshLuedeman/teamwork", "Source repository (owner/repo)")
	installCmd.Flags().String("ref", "main", "Git ref to install from (branch, tag, or SHA)")
	installCmd.Flags().Bool("force", false, "Overwrite existing installation")
}

func runInstall(cmd *cobra.Command, args []string) error {
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

	owner, repo, err := parseSource(source)
	if err != nil {
		return err
	}

	// --force removes the version file so Install doesn't refuse to run.
	if force {
		os.Remove(filepath.Join(dir, ".teamwork", "framework-version.txt"))
	}
	return installer.Install(dir, owner, repo, ref)
}

func parseSource(source string) (string, string, error) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --source format %q: expected owner/repo", source)
	}
	return parts[0], parts[1], nil
}
