package cmd

import (
	"fmt"
	"strings"

	"github.com/JoshLuedeman/teamwork/internal/config"
	gh "github.com/JoshLuedeman/teamwork/internal/github"
	"github.com/JoshLuedeman/teamwork/internal/installer"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Teamwork framework files to the latest version",
	Long: `Update fetches the latest framework files from the upstream Teamwork repository
and applies changes. Files that have been modified locally are skipped with a
warning unless --force is set.

By default, if agent files with unfilled placeholders are detected and the gh CLI
is available, an issue is created and assigned to Copilot to run /setup-teamwork.
Use --create-issue=false to disable this behavior.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().String("source", "JoshLuedeman/teamwork", "Source repository (owner/repo)")
	updateCmd.Flags().String("ref", "main", "Git ref to update to (branch, tag, or SHA)")
	updateCmd.Flags().Bool("force", false, "Overwrite user-modified files without warning")
	updateCmd.Flags().Bool("create-issue", true, "Create a GitHub issue assigned to Copilot for setup when placeholders are detected")
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
	createIssue, err := cmd.Flags().GetBool("create-issue")
	if err != nil {
		return err
	}

	owner, repo, err := parseUpdateSource(source)
	if err != nil {
		return err
	}

	if err := installer.Update(dir, owner, repo, ref, force); err != nil {
		return err
	}

	// After a successful update, check for unfilled placeholders and
	// optionally create a GitHub issue assigned to Copilot.
	if createIssue {
		maybeCreateSetupIssue(dir)
	}

	return nil
}

// setupIssueTitle is the canonical title used for setup-teamwork issues.
// It is also used to detect duplicates.
const setupIssueTitle = "[TASK] Run /setup-teamwork to configure agent files"

// setupIssueLabel is applied to setup issues for easy querying.
const setupIssueLabel = "setup"

// maybeCreateSetupIssue creates a GitHub issue assigned to Copilot if
// unfilled CUSTOMIZE placeholders are detected in agent files. It checks
// for existing open issues with the same title to avoid duplicates.
// Failures are non-fatal — the update already succeeded.
func maybeCreateSetupIssue(dir string) {
	files := installer.CustomizePlaceholderFiles(dir)
	if len(files) == 0 {
		return
	}

	cfg, err := config.Load(dir)
	if err != nil {
		return // No config → can't determine target repo.
	}

	// Skip if config still has the default placeholder repo.
	if cfg.Project.Repo == "" || cfg.Project.Repo == "owner/repo" {
		return
	}

	client, err := gh.NewClientFromConfig(cfg)
	if err != nil {
		return
	}
	if !client.Available() {
		return // gh CLI not installed.
	}

	// Check for an existing open setup issue to avoid duplicates.
	existing, err := client.ListIssues("open", []string{setupIssueLabel})
	if err == nil {
		for _, iss := range existing {
			if iss.Title == setupIssueTitle {
				fmt.Printf("  Setup issue already exists: #%d — skipping issue creation.\n", iss.Number)
				return
			}
		}
	}

	body := buildSetupIssueBody(files)
	issueNum, err := client.CreateIssue(setupIssueTitle, body, []string{setupIssueLabel}, []string{"copilot"})
	if err != nil {
		fmt.Printf("  Warning: could not create setup issue: %v\n", err)
		return
	}
	fmt.Printf("  Created setup issue #%d assigned to Copilot.\n", issueNum)
}

// buildSetupIssueBody constructs the Markdown body for the setup issue.
func buildSetupIssueBody(files []string) string {
	var b strings.Builder
	b.WriteString("## Setup Teamwork Agent Files\n\n")
	b.WriteString("The Teamwork framework has been updated. The following agent files still have\n")
	b.WriteString("unfilled `<!-- CUSTOMIZE -->` placeholders that need to be configured for this project:\n\n")
	for _, f := range files {
		fmt.Fprintf(&b, "- `.github/agents/%s`\n", f)
	}
	b.WriteString("\n### Instructions\n\n")
	b.WriteString("Run the `/setup-teamwork` skill to auto-detect this project's tech stack and fill in the placeholders.\n\n")
	b.WriteString("This will:\n")
	b.WriteString("1. Scan the repository for config files (`package.json`, `go.mod`, `pyproject.toml`, etc.)\n")
	b.WriteString("2. Detect languages, frameworks, test tools, and build commands\n")
	b.WriteString("3. Fill in the `<!-- CUSTOMIZE -->` placeholders with detected values\n")
	b.WriteString("4. Report any remaining placeholders that need manual input\n")
	return b.String()
}

func parseUpdateSource(source string) (string, string, error) {
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --source format %q: expected owner/repo", source)
	}
	return parts[0], parts[1], nil
}
