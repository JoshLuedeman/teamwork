package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joshluedeman/teamwork/internal/validate"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment and project health",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name   string
	status string // "ok", "warn", "fail"
	detail string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	var results []checkResult

	// 1. Check .teamwork/ directory
	results = append(results, checkTeamworkDir(dir))

	// 2. Validate config.yaml
	results = append(results, checkConfig(dir))

	// 3. Git installed and configured
	results = append(results, checkGit()...)

	// 4. AI tools available
	results = append(results, checkAITools()...)

	// 5. GitHub CLI
	results = append(results, checkGitHubCLI()...)

	// 6. GH_TOKEN
	results = append(results, checkGHToken())

	// 7. Go toolchain
	results = append(results, checkGo())

	// Print results
	failCount := 0
	warnCount := 0
	for _, r := range results {
		switch r.status {
		case "ok":
			fmt.Printf("[✓] %s\n", r.detail)
		case "warn":
			fmt.Printf("[!] %s\n", r.detail)
			warnCount++
		case "fail":
			fmt.Printf("[✗] %s\n", r.detail)
			failCount++
		}
	}

	fmt.Println()
	if failCount > 0 {
		return &ExitError{Code: 1, Message: fmt.Sprintf("%d issue(s) found, %d warning(s)", failCount, warnCount)}
	} else if warnCount > 0 {
		fmt.Printf("No issues found, %d warning(s)\n", warnCount)
	} else {
		fmt.Println("Everything looks good!")
	}

	return nil
}

func checkTeamworkDir(dir string) checkResult {
	teamworkDir := filepath.Join(dir, ".teamwork")
	info, err := os.Stat(teamworkDir)
	if err != nil || !info.IsDir() {
		return checkResult{"teamwork-dir", "fail", ".teamwork/ directory not found — run 'teamwork init'"}
	}

	// Check expected subdirectories
	subdirs := []string{"state", "handoffs", "memory", "metrics"}
	missing := []string{}
	for _, sub := range subdirs {
		p := filepath.Join(teamworkDir, sub)
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			missing = append(missing, sub)
		}
	}

	if len(missing) > 0 {
		return checkResult{"teamwork-dir", "warn", fmt.Sprintf(".teamwork/ exists but missing subdirectories: %v", missing)}
	}

	return checkResult{"teamwork-dir", "ok", ".teamwork/ directory initialized"}
}

func checkConfig(dir string) checkResult {
	results, err := validate.Run(dir)
	if err != nil {
		return checkResult{"config", "fail", fmt.Sprintf("Cannot validate config: %v", err)}
	}

	failed := 0
	for _, r := range results {
		if !r.Passed {
			failed++
		}
	}

	if failed > 0 {
		return checkResult{"config", "fail", fmt.Sprintf("config.yaml has %d validation error(s) — run 'teamwork validate' for details", failed)}
	}

	return checkResult{"config", "ok", "config.yaml valid"}
}

func checkGit() []checkResult {
	var results []checkResult

	// Check git is installed
	if _, err := exec.LookPath("git"); err != nil {
		results = append(results, checkResult{"git", "fail", "Git not found — install git"})
		return results
	}

	// Check user.name
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil || len(out) == 0 {
		results = append(results, checkResult{"git-name", "warn", "Git user.name not set — run 'git config user.name \"Your Name\"'"})
	} else {
		results = append(results, checkResult{"git-name", "ok", fmt.Sprintf("Git configured (user: %s)", trimOutput(out))})
	}

	// Check user.email
	out, err = exec.Command("git", "config", "user.email").Output()
	if err != nil || len(out) == 0 {
		results = append(results, checkResult{"git-email", "warn", "Git user.email not set — run 'git config user.email \"you@example.com\"'"})
	} else {
		results = append(results, checkResult{"git-email", "ok", fmt.Sprintf("Git email configured (%s)", trimOutput(out))})
	}

	return results
}

func checkAITools() []checkResult {
	var results []checkResult

	tools := []struct {
		name    string
		binary  string
		install string
	}{
		{"Claude CLI", "claude", "npm install -g @anthropic-ai/claude-code"},
		{"GitHub Copilot CLI", "github-copilot-cli", "npm install -g @githubnext/github-copilot-cli"},
	}

	found := false
	for _, tool := range tools {
		if _, err := exec.LookPath(tool.binary); err == nil {
			results = append(results, checkResult{tool.name, "ok", fmt.Sprintf("%s available", tool.name)})
			found = true
		}
	}

	if !found {
		results = append(results, checkResult{"ai-tools", "warn", "No AI CLI tools found (claude, github-copilot-cli) — agents may not be invokable locally"})
	}

	return results
}

func checkGitHubCLI() []checkResult {
	var results []checkResult

	if _, err := exec.LookPath("gh"); err != nil {
		results = append(results, checkResult{"gh-cli", "warn", "GitHub CLI (gh) not found — install from https://cli.github.com"})
		return results
	}

	// Check if authenticated
	err := exec.Command("gh", "auth", "status").Run()
	if err != nil {
		results = append(results, checkResult{"gh-auth", "warn", "GitHub CLI not authenticated — run 'gh auth login'"})
	} else {
		results = append(results, checkResult{"gh-auth", "ok", "GitHub CLI authenticated"})
	}

	return results
}

func checkGHToken() checkResult {
	if os.Getenv("GH_TOKEN") != "" || os.Getenv("GITHUB_TOKEN") != "" {
		return checkResult{"gh-token", "ok", "GH_TOKEN or GITHUB_TOKEN set"}
	}
	return checkResult{"gh-token", "warn", "GH_TOKEN not set — required for private repos"}
}

func checkGo() checkResult {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return checkResult{"go", "warn", "Go not found — required for building from source"}
	}
	return checkResult{"go", "ok", fmt.Sprintf("Go available (%s)", trimOutput(out))}
}

func trimOutput(b []byte) string {
	s := string(b)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}
