package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/spf13/cobra"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List configured repositories",
	RunE:  runRepos,
}

func init() {
	rootCmd.AddCommand(reposCmd)
}

func runRepos(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Repos) == 0 {
		fmt.Println("No repositories configured.")
		fmt.Println("Add a repos section to .teamwork/config.yaml to enable multi-repo coordination.")
		return nil
	}

	fmt.Printf("%-16s  %-30s  %-10s  %s\n", "Name", "Repo", "Status", "Path")
	fmt.Println("----------------  ------------------------------  ----------  --------------------")

	for _, r := range cfg.Repos {
		path := r.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}

		status := repoStatus(path)
		fmt.Printf("%-16s  %-30s  %-10s  %s\n", r.Name, r.Repo, status, r.Path)
	}

	return nil
}

func repoStatus(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "not found"
	}
	if !info.IsDir() {
		return "not a dir"
	}

	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return "no git"
	}

	// Check for uncommitted changes.
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "git error"
	}
	if len(out) > 0 {
		return "dirty"
	}
	return "clean"
}
