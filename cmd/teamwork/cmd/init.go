package cmd

import (
"bufio"
"fmt"
"io"
"os"
"path/filepath"
"strings"

"github.com/joshluedeman/teamwork/internal/config"
"github.com/joshluedeman/teamwork/internal/installer"
"github.com/joshluedeman/teamwork/internal/presets"
"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
Use:   "init",
Short: "Initialize a Teamwork project: fetch framework files and create config",
Long: `Init sets up a complete Teamwork project in two steps:

1. Fetches framework files (agents, skills, docs, instructions) from the upstream
   Teamwork repository — equivalent to the former 'teamwork install' command.
2. Creates the .teamwork/ directory with config, memory seeds, and subdirectories.

If framework files are already installed, the fetch step is skipped and only the
config is created. Use --force to re-fetch framework files.`,
RunE: runInit,
}

func init() {
initCmd.Flags().Bool("non-interactive", false, "Skip interactive wizard even when stdin is a TTY")
initCmd.Flags().String("preset", "", "Use a preset config for a specific stack ("+strings.Join(presets.Names(), ", ")+")")
initCmd.Flags().String("source", "joshluedeman/teamwork", "Source repository for framework files (owner/repo)")
initCmd.Flags().String("ref", "main", "Git ref to install from (branch, tag, or SHA)")
initCmd.Flags().Bool("force", false, "Re-fetch framework files even if already installed")
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

presetName, err := cmd.Flags().GetString("preset")
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

// Step 1: Fetch framework files from upstream (agents, skills, docs, etc.).
owner, repo, err := parseSource(source)
if err != nil {
return err
}

if force {
os.Remove(filepath.Join(dir, ".teamwork", "framework-version.txt"))
}

if installer.IsInstalled(dir) {
fmt.Println("Framework files already installed — skipping fetch.")
} else {
if err := installer.Install(dir, owner, repo, ref); err != nil {
return fmt.Errorf("installing framework files: %w", err)
}
}

// Step 2: Create .teamwork/ config and memory seeds (if not already present).
teamworkDir := filepath.Join(dir, ".teamwork")
configPath := filepath.Join(teamworkDir, "config.yaml")

// If config already exists, we're done.
if _, err := os.Stat(configPath); err == nil {
fmt.Println("Config already exists — initialization complete.")
return nil
}

var cfg *config.Config
if presetName != "" {
cfg, err = presets.Get(presetName)
if err != nil {
return err
}
fmt.Printf("Using preset: %s\n", presetName)
} else {
cfg = config.Default()
}

// Interactive prompts when stdin is a TTY and not explicitly disabled.
if !nonInteractive && isInteractive() {
cfg = runWizard(cfg, os.Stdin)
}

// Override quality gate defaults based on what the project actually has.
cfg.QualityGates.TestsPass = config.DetectTestFramework(dir)
cfg.QualityGates.LintPass = config.DetectLinter(dir)

// Create subdirectories (installer.Install already creates these, but
// ensure they exist in case framework was pre-installed without them).
subdirs := []string{"state", "handoffs", "memory", "metrics"}
for _, sub := range subdirs {
if err := os.MkdirAll(filepath.Join(teamworkDir, sub), 0o755); err != nil {
return fmt.Errorf("creating %s: %w", sub, err)
}
}

if err := cfg.Save(dir); err != nil {
return fmt.Errorf("writing config: %w", err)
}

// Create seeded memory files with example entries.
for name, content := range seedMemoryFiles() {
path := filepath.Join(teamworkDir, "memory", name)
if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
return fmt.Errorf("creating %s: %w", name, err)
}
}

fmt.Println("Initialized .teamwork/ directory.")
fmt.Printf("  Config: %s\n", filepath.Join(teamworkDir, "config.yaml"))
fmt.Printf("  Project: %s (%s)\n", cfg.Project.Name, cfg.Project.Repo)
return nil
}

func parseSource(source string) (string, string, error) {
parts := strings.SplitN(source, "/", 2)
if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
return "", "", fmt.Errorf("invalid --source format %q: expected owner/repo", source)
}
return parts[0], parts[1], nil
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

// seedMemoryFiles returns a map of filename to seed content for each memory
// file. Each file contains 1–2 example entries that demonstrate the YAML
// structure with all available fields, clearly marked as examples.
func seedMemoryFiles() map[string]string {
return map[string]string{
"patterns.yaml": `# Patterns That Work
#
# Approaches that work well in this project. Agents should repeat these.
# Add entries as you discover what works. See docs/protocols.md for format.
#
# Available fields per entry:
#   id:      Unique identifier (e.g. pattern-001). Auto-generated if omitted.
#   date:    ISO 8601 date (e.g. 2025-01-15). Defaults to today if omitted.
#   source:  Where this was discovered (e.g. "PR #42 review", "incident retro").
#   domain:  List of topic tags for indexing (e.g. ["auth", "api"]).
#   content: The pattern itself — what to do.
#   context: Why this works, when it was discovered, or supporting details.

entries:
  - id: pattern-001
    date: "2025-01-01"
    source: "example"
    domain:
      - example
    content: "This is an example pattern entry — replace or delete it"
    context: "Demonstrates the format for pattern entries with all available fields"
`,

"antipatterns.yaml": `# Anti-Patterns
#
# Approaches that failed or caused problems. Agents should avoid these.
# Add entries when you discover what doesn't work. See docs/protocols.md for format.
#
# Available fields per entry:
#   id:      Unique identifier (e.g. antipattern-001). Auto-generated if omitted.
#   date:    ISO 8601 date (e.g. 2025-01-15). Defaults to today if omitted.
#   source:  Where this was discovered (e.g. "incident retrospective").
#   domain:  List of topic tags for indexing (e.g. ["deployment", "testing"]).
#   content: The anti-pattern itself — what to avoid.
#   context: Why this failed, what happened, or how it was discovered.

entries:
  - id: antipattern-001
    date: "2025-01-01"
    source: "example"
    domain:
      - example
    content: "This is an example anti-pattern entry — replace or delete it"
    context: "Demonstrates the format for anti-pattern entries with all available fields"
`,

"decisions.yaml": `# Key Decisions
#
# Significant decisions with rationale and date. Complements ADRs with
# lighter-weight entries. See docs/protocols.md for format.
#
# Available fields per entry:
#   id:      Unique identifier (e.g. decision-001). Auto-generated if omitted.
#   date:    ISO 8601 date (e.g. 2025-01-15). Defaults to today if omitted.
#   source:  Where this decision was made (e.g. "architecture discussion").
#   domain:  List of topic tags for indexing (e.g. ["architecture", "auth"]).
#   content: The decision itself — what was decided.
#   context: Why this decision was made, alternatives considered, or tradeoffs.

entries:
  - id: decision-001
    date: "2025-01-01"
    source: "example"
    domain:
      - example
    content: "This is an example decision entry — replace or delete it"
    context: "Demonstrates the format for decision entries with all available fields"
`,

"feedback.yaml": `# Reviewer and Human Feedback
#
# Broadly applicable feedback from code reviews and human input.
# Not PR-specific — these are lessons that apply across the project.
# See docs/protocols.md for format.
#
# Available fields per entry:
#   id:      Unique identifier (e.g. feedback-001). Auto-generated if omitted.
#   date:    ISO 8601 date (e.g. 2025-01-15). Defaults to today if omitted.
#   source:  Where this feedback came from (e.g. "PR #42 review").
#   domain:  List of topic tags for indexing (e.g. ["testing", "error-handling"]).
#   content: The feedback itself — the lesson learned.
#   context: Broader context or why this feedback matters.

entries:
  - id: feedback-001
    date: "2025-01-01"
    source: "example"
    domain:
      - example
    content: "This is an example feedback entry — replace or delete it"
    context: "Demonstrates the format for feedback entries with all available fields"
`,

"index.yaml": `# Memory Index
#
# Maps domains/topics to entry IDs across all memory files for fast lookup.
# This file is updated automatically when entries are added via the CLI.
# See docs/protocols.md for format.
#
# Structure:
#   domains:
#     <domain-name>:
#       - <entry-id-1>
#       - <entry-id-2>

domains:
  example:
    - pattern-001
    - antipattern-001
    - decision-001
    - feedback-001
`,
}
}