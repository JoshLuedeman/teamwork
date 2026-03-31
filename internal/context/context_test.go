package agentcontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/state"
)

func writeTestConfig(t *testing.T, dir, yaml string) {
	t.Helper()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

const minimalConfig = `project:
  name: test
  repo: test/test
roles:
  core: [planner, architect, coder, tester, reviewer, security-auditor, documenter]
quality_gates:
  handoff_complete: true
  tests_pass: true
  lint_pass: true
`

func writeRoleFile(t *testing.T, dir, role, content string) {
	t.Helper()
	p := filepath.Join(dir, ".github", "agents", role+".agent.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMemoryFile(t *testing.T, dir, name, content string) {
	t.Helper()
	memDir := filepath.Join(dir, ".teamwork", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssemble_ContainsRoleFile(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/1-test", "feature", "Implement authentication")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	roleContent := "# Coder Agent\n\nYou implement features."
	writeRoleFile(t, dir, "coder", roleContent)

	pkg, err := Assemble(dir, "feature/1-test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pkg.Role != "coder" {
		t.Errorf("expected role=coder, got %s", pkg.Role)
	}

	if !strings.Contains(pkg.RoleFile, "Coder Agent") {
		t.Errorf("expected role file content in package, got: %q", pkg.RoleFile)
	}

	rendered := pkg.Render()
	if !strings.Contains(rendered, "Coder Agent") {
		t.Errorf("expected rendered output to contain role file content:\n%s", rendered)
	}
	if !strings.Contains(rendered, "## Role Definition") {
		t.Errorf("expected '## Role Definition' section in output:\n%s", rendered)
	}
}

func TestAssemble_MissingRoleFileGraceful(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/2-missing-role", "feature", "Build search feature")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	// Do NOT write a role file — test graceful fallback.
	pkg, err := Assemble(dir, "feature/2-missing-role", 0)
	if err != nil {
		t.Fatalf("expected no error for missing role file, got: %v", err)
	}

	if pkg.RoleFile != "" {
		t.Errorf("expected empty RoleFile for missing file, got: %q", pkg.RoleFile)
	}

	// Render should not crash.
	rendered := pkg.Render()
	if rendered == "" {
		t.Error("expected non-empty rendered output even with missing role file")
	}
	if !strings.Contains(rendered, "role file not found") {
		t.Errorf("expected fallback message in rendered output:\n%s", rendered)
	}
}

func TestAssemble_TokenEstimateNonZero(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/3-token", "feature", "Implement token estimation test")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	writeRoleFile(t, dir, "coder", "# Coder\nThis is the coder role with some content to ensure token estimate is non-zero.")

	pkg, err := Assemble(dir, "feature/3-token", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pkg.TokenEstimate <= 0 {
		t.Errorf("expected positive token estimate, got %d", pkg.TokenEstimate)
	}
}

func TestAssemble_WithMemoryItems(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/4-memory", "feature", "authentication implementation")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	writeMemoryFile(t, dir, "patterns.yaml", `entries:
  - id: pattern-001
    domain: [feature]
    content: "Always use bcrypt for password hashing in authentication"
    context: "Applied in auth module"
`)

	pkg, err := Assemble(dir, "feature/4-memory", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pkg.MemoryItems) == 0 {
		t.Error("expected memory items to be populated")
	}

	rendered := pkg.Render()
	if !strings.Contains(rendered, "## Relevant Memory") {
		t.Errorf("expected '## Relevant Memory' section:\n%s", rendered)
	}
}

func TestAssemble_StepZeroUsesCurrentStep(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/5-step", "feature", "Test step resolution")
	ws.CurrentStep = 5
	ws.CurrentRole = "tester"
	ws.Steps = []state.StepRecord{
		{Step: 5, Role: "tester", Action: "Validate", Started: "2024-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	pkg, err := Assemble(dir, "feature/5-step", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pkg.Step != 5 {
		t.Errorf("expected step=5, got %d", pkg.Step)
	}
	if pkg.Role != "tester" {
		t.Errorf("expected role=tester, got %s", pkg.Role)
	}
}
