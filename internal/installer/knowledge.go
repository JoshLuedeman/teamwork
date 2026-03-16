package installer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joshluedeman/teamwork/internal/config"
)

// placeholderRE matches bracketed example placeholders of the form [e.g., ...].
var placeholderRE = regexp.MustCompile(`\[e\.g\.,\s*[^\]]+\]`)

// PopulateProjectKnowledge detects the project tech stack and replaces
// [e.g., ...] placeholder values in the Project Knowledge section of each
// .github/agents/*.agent.md file. Only non-empty detected values are applied;
// unrecognised fields are left unchanged.
func PopulateProjectKnowledge(dir string) {
	info := config.Detect(dir)

	// Map from the bold field label in the agent file to the detected value.
	replacements := map[string]string{
		"**Tech Stack:**":          info.TechStack,
		"**Languages:**":           info.Languages,
		"**Package Manager:**":     info.PackageManager,
		"**Dependency Manifest:**": info.DependencyManifest,
		"**Lockfile:**":            info.Lockfile,
		"**Test Framework:**":      info.TestFramework,
		"**Build Command:**":       info.BuildCommand,
		"**Test Command:**":        info.TestCommand,
		"**Lint Command:**":        info.LintCommand,
		"**Audit Command:**":       info.AuditCommand,
	}

	agentsDir := filepath.Join(dir, ".github", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".agent.md") {
			continue
		}
		path := filepath.Join(agentsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		updated := applyKnowledgeReplacements(string(data), replacements)
		if updated != string(data) {
			_ = os.WriteFile(path, []byte(updated), 0o644)
		}
	}
}

// applyKnowledgeReplacements replaces [e.g., ...] placeholders in content
// based on field context. Only lines that contain both a known field label
// and a placeholder are modified; all other lines are returned unchanged.
func applyKnowledgeReplacements(content string, replacements map[string]string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !placeholderRE.MatchString(line) {
			continue
		}
		for field, value := range replacements {
			if value == "" {
				continue
			}
			if strings.Contains(line, field) {
				lines[i] = placeholderRE.ReplaceAllLiteralString(line, value)
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}
