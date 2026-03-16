package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyKnowledgeReplacements_ReplacesMatchingFields(t *testing.T) {
	content := "## Project Knowledge\n" +
		"<!-- CUSTOMIZE: Replace the placeholders below with your project's details -->\n" +
		"- **Tech Stack:** [e.g., React 18, TypeScript, Node.js 20, PostgreSQL 16]\n" +
		"- **Languages:** [e.g., TypeScript, Go, Python]\n" +
		"- **Package Manager:** [e.g., npm, pnpm, yarn, go mod]\n" +
		"- **Test Framework:** [e.g., Jest, pytest, go test]\n" +
		"- **Build Command:** [e.g., `npm run build`, `make build`]\n" +
		"- **Test Command:** [e.g., `npm test`, `make test`]\n" +
		"- **Lint Command:** [e.g., `npm run lint`, `golangci-lint run`]\n"

	replacements := map[string]string{
		"**Tech Stack:**":      "Go 1.24",
		"**Languages:**":       "Go 1.24",
		"**Package Manager:**": "go mod",
		"**Test Framework:**":  "go test",
		"**Build Command:**":   "go build ./...",
		"**Test Command:**":    "go test ./...",
		"**Lint Command:**":    "golangci-lint run",
	}
	got := applyKnowledgeReplacements(content, replacements)

	cases := []string{
		"- **Tech Stack:** Go 1.24",
		"- **Languages:** Go 1.24",
		"- **Package Manager:** go mod",
		"- **Test Framework:** go test",
		"- **Build Command:** go build ./...",
		"- **Test Command:** go test ./...",
		"- **Lint Command:** golangci-lint run",
	}
	for _, want := range cases {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q\ngot:\n%s", want, got)
		}
	}
}

func TestApplyKnowledgeReplacements_EmptyValueSkipped(t *testing.T) {
	content := "- **Tech Stack:** [e.g., React 18]\n"
	replacements := map[string]string{
		"**Tech Stack:**": "", // empty — should not replace
	}
	got := applyKnowledgeReplacements(content, replacements)
	if got != content {
		t.Errorf("expected content unchanged when value is empty\ngot: %q\nwant: %q", got, content)
	}
}

func TestApplyKnowledgeReplacements_UnknownFieldUnchanged(t *testing.T) {
	content := "- **Custom Field:** [e.g., something special]\n"
	replacements := map[string]string{
		"**Tech Stack:**": "Go 1.24",
	}
	got := applyKnowledgeReplacements(content, replacements)
	if got != content {
		t.Errorf("expected unknown field to remain unchanged\ngot: %q\nwant: %q", got, content)
	}
}

func TestApplyKnowledgeReplacements_NoPlaceholderUnchanged(t *testing.T) {
	content := "- **Languages:** Go (already set)\n"
	replacements := map[string]string{
		"**Languages:**": "Go 1.24",
	}
	got := applyKnowledgeReplacements(content, replacements)
	if got != content {
		t.Errorf("expected line without placeholder to remain unchanged\ngot: %q\nwant: %q", got, content)
	}
}

func TestPopulateProjectKnowledge_GoProject(t *testing.T) {
	dir := t.TempDir()

	// Write a go.mod so Detect() identifies a Go project.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create agents directory with a sample agent file.
	agentsDir := filepath.Join(dir, ".github", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentContent := "---\nname: coder\n---\n\n## Project Knowledge\n" +
		"<!-- CUSTOMIZE: Replace the placeholders below with your project's details -->\n" +
		"- **Languages:** [e.g., TypeScript, Go, Python]\n" +
		"- **Package Manager:** [e.g., npm, pnpm, yarn, go mod]\n" +
		"- **Test Command:** [e.g., `npm test`, `make test`]\n" +
		"- **Lint Command:** [e.g., `npm run lint`, `golangci-lint run`]\n"

	if err := os.WriteFile(filepath.Join(agentsDir, "coder.agent.md"), []byte(agentContent), 0o644); err != nil {
		t.Fatal(err)
	}

	PopulateProjectKnowledge(dir)

	data, err := os.ReadFile(filepath.Join(agentsDir, "coder.agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, "- **Languages:** Go 1.24") {
		t.Errorf("expected Languages to be populated with 'Go 1.24'\ngot:\n%s", got)
	}
	if !strings.Contains(got, "- **Package Manager:** go mod") {
		t.Errorf("expected Package Manager to be populated with 'go mod'\ngot:\n%s", got)
	}
	if !strings.Contains(got, "- **Test Command:** go test ./...") {
		t.Errorf("expected Test Command to be populated\ngot:\n%s", got)
	}
	if !strings.Contains(got, "- **Lint Command:** golangci-lint run") {
		t.Errorf("expected Lint Command to be populated\ngot:\n%s", got)
	}
}

func TestPopulateProjectKnowledge_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	// Should not panic when .github/agents/ doesn't exist.
	PopulateProjectKnowledge(dir)
}

func TestPopulateProjectKnowledge_UnknownProject_PlaceholdersUnchanged(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".github", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := "- **Languages:** [e.g., TypeScript, Go, Python]\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "coder.agent.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	PopulateProjectKnowledge(dir)

	data, _ := os.ReadFile(filepath.Join(agentsDir, "coder.agent.md"))
	if string(data) != original {
		t.Errorf("expected placeholder unchanged for unknown project\ngot: %q\nwant: %q", string(data), original)
	}
}

func TestPopulateProjectKnowledge_NonAgentFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentsDir := filepath.Join(dir, ".github", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Non-.agent.md file should be ignored.
	original := "- **Languages:** [e.g., TypeScript, Go, Python]\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "README.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	PopulateProjectKnowledge(dir)

	data, _ := os.ReadFile(filepath.Join(agentsDir, "README.md"))
	if string(data) != original {
		t.Errorf("non-.agent.md file should not be modified\ngot: %q\nwant: %q", string(data), original)
	}
}
