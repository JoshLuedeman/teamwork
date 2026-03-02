package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JoshLuedeman/teamwork/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .teamwork/ directory in the current project",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	teamworkDir := filepath.Join(dir, ".teamwork")

	if _, err := os.Stat(teamworkDir); err == nil {
		fmt.Println(".teamwork/ already exists — skipping initialization.")
		return nil
	}

	subdirs := []string{"state", "handoffs", "memory", "metrics"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", sub, err)
		}
	}

	cfg := config.Default()
	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}

	memoryFiles := []string{
		"patterns.yaml",
		"antipatterns.yaml",
		"decisions.yaml",
		"feedback.yaml",
		"index.yaml",
	}
	for _, name := range memoryFiles {
		path := filepath.Join(teamworkDir, "memory", name)
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			return fmt.Errorf("creating %s: %w", name, err)
		}
	}

	fmt.Println("Initialized .teamwork/ directory.")
	return nil
}
