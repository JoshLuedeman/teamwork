package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .teamwork/ directory in the current project",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("non-interactive", false, "Skip interactive wizard even when stdin is a TTY")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		return err
	}

	teamworkDir := filepath.Join(dir, ".teamwork")

	if _, err := os.Stat(teamworkDir); err == nil {
		fmt.Println(".teamwork/ already exists \u2014 skipping initialization.")
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking .teamwork/: %w", err)
	}

	cfg := config.Default()

	// Interactive prompts when stdin is a TTY and not explicitly disabled.
	if !nonInteractive && isInteractive() {
		cfg = runWizard(cfg, os.Stdin)
	}

	// Create subdirectories.
	subdirs := []string{"state", "handoffs", "memory", "metrics"}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", sub, err)
		}
	}

	if err := cfg.Save(dir); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Create empty memory files.
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
	fmt.Printf("  Config: %s\n", filepath.Join(teamworkDir, "config.yaml"))
	fmt.Printf("  Project: %s (%s)\n", cfg.Project.Name, cfg.Project.Repo)
	return nil
}

// isInteractive reports whether stdin is connected to a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// runWizard prompts the user for project settings, falling back to the
// defaults already present in cfg when the user presses Enter.
func runWizard(cfg *config.Config, r io.Reader) *config.Config {
	reader := bufio.NewReader(r)

	fmt.Println("Teamwork Setup Wizard")
	fmt.Println("Press Enter to accept defaults shown in [brackets].")
	fmt.Println()

	// Project name.
	fmt.Printf("Project name [%s]: ", cfg.Project.Name)
	if name := readLine(reader); name != "" {
		cfg.Project.Name = name
	}

	// GitHub repo.
	fmt.Printf("GitHub repo (owner/repo) [%s]: ", cfg.Project.Repo)
	if repo := readLine(reader); repo != "" {
		cfg.Project.Repo = repo
	}

	// Optional roles.
	fmt.Print("Enable optional roles? (triager, devops, dependency-manager, refactorer) [y/N]: ")
	if answer := readLine(reader); strings.HasPrefix(strings.ToLower(answer), "y") {
		cfg.Roles.Optional = []string{"triager", "devops", "dependency-manager", "refactorer"}
	}

	fmt.Println()
	return cfg
}

// readLine reads a single line from the reader and trims whitespace.
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
