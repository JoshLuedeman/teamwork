package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// resetStartFlags resets the start command flags between tests to avoid
// pollution from previous test runs.
func resetStartFlags(t *testing.T) {
	t.Helper()
	startCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeStartCmd runs "teamwork start" with the given args and captures stdout.
func executeStartCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	resetStartFlags(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"start", "--dir", dir}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

// minimalConfig is the smallest valid config for dry-run tests.
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

// skipStepsConfig has skip_steps configured for documentation and spike.
const skipStepsConfig = `project:
  name: test
  repo: test/test
roles:
  core: [planner, architect, coder, tester, reviewer, security-auditor, documenter]
workflows:
  skip_steps:
    documentation: [security-auditor]
    spike: [tester, security-auditor]
quality_gates:
  handoff_complete: true
  tests_pass: true
  lint_pass: true
`

func TestDryRun_NoStateFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	stateDir := filepath.Join(dir, ".teamwork", "state")

	out, err := executeStartCmd(t, dir, "feature", "Add OAuth", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify no state directory was created.
	if _, statErr := os.Stat(stateDir); statErr == nil {
		entries, _ := os.ReadDir(stateDir)
		t.Errorf("expected no state files, but state dir exists with %d entries", len(entries))
	}
}

func TestDryRun_FeatureWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	out, err := executeStartCmd(t, dir, "feature", "Add OAuth", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Check header.
	if !strings.Contains(out, "Workflow: feature") {
		t.Errorf("expected 'Workflow: feature' in output:\n%s", out)
	}
	if !strings.Contains(out, "Goal: Add OAuth") {
		t.Errorf("expected 'Goal: Add OAuth' in output:\n%s", out)
	}

	// Check all feature step roles are present.
	for _, want := range []string{
		"Human",
		"Planner",
		"Architect",
		"Coder",
		"Tester",
		"Security Auditor",
		"Reviewer",
		"Documenter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected role %q in output:\n%s", want, out)
		}
	}

	// Check model tiers are shown.
	if !strings.Contains(out, "(premium)") {
		t.Errorf("expected '(premium)' tier in output:\n%s", out)
	}
	if !strings.Contains(out, "(standard)") {
		t.Errorf("expected '(standard)' tier in output:\n%s", out)
	}
	if !strings.Contains(out, "(fast)") {
		t.Errorf("expected '(fast)' tier in output:\n%s", out)
	}

	// Check quality gates.
	if !strings.Contains(out, "Quality gates: handoff_complete, tests_pass, lint_pass") {
		t.Errorf("expected quality gates line in output:\n%s", out)
	}

	// No skipped steps for feature with minimal config.
	if !strings.Contains(out, "Skipped steps: none") {
		t.Errorf("expected 'Skipped steps: none' in output:\n%s", out)
	}

	// Total agent steps: 7 (9 steps minus 2 human steps).
	if !strings.Contains(out, "Total agent steps: 7") {
		t.Errorf("expected 'Total agent steps: 7' in output:\n%s", out)
	}
}

func TestDryRun_DocumentationWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	out, err := executeStartCmd(t, dir, "documentation", "Update API docs", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Workflow: documentation") {
		t.Errorf("expected 'Workflow: documentation' in output:\n%s", out)
	}

	// Documentation has 5 steps: human, documenter, documenter, reviewer, human.
	if !strings.Contains(out, "Documenter") {
		t.Errorf("expected 'Documenter' in output:\n%s", out)
	}
	if !strings.Contains(out, "Reviewer") {
		t.Errorf("expected 'Reviewer' in output:\n%s", out)
	}

	// 3 agent steps (2 documenter + 1 reviewer).
	if !strings.Contains(out, "Total agent steps: 3") {
		t.Errorf("expected 'Total agent steps: 3' in output:\n%s", out)
	}
}

func TestDryRun_SkipSteps(t *testing.T) {
	// Use a config that skips security-auditor for feature workflows.
	skipFeatureConfig := `project:
  name: test
  repo: test/test
roles:
  core: [planner, architect, coder, tester, reviewer, security-auditor, documenter]
workflows:
  skip_steps:
    feature: [security-auditor]
quality_gates:
  handoff_complete: true
  tests_pass: true
  lint_pass: true
`
	dir := t.TempDir()
	writeTestConfig(t, dir, skipFeatureConfig)

	out, err := executeStartCmd(t, dir, "feature", "Add OAuth", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "SKIPPED") {
		t.Errorf("expected 'SKIPPED' marker in output:\n%s", out)
	}
	if !strings.Contains(out, "Skipped steps: security-auditor") {
		t.Errorf("expected 'Skipped steps: security-auditor' in output:\n%s", out)
	}
	// Total agent steps should be 6 (7 minus the skipped security-auditor).
	if !strings.Contains(out, "Total agent steps: 6") {
		t.Errorf("expected 'Total agent steps: 6' in output:\n%s", out)
	}
}

func TestDryRun_UnknownWorkflowType(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	_, err := executeStartCmd(t, dir, "nonexistent", "some goal", "--dry-run")
	if err == nil {
		t.Fatal("expected error for unknown workflow type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown workflow type") {
		t.Errorf("expected 'unknown workflow type' error, got: %v", err)
	}
}

func TestStart_NormalStartStillWorks(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	// Normal start (without --dry-run) should create state files.
	out, err := executeStartCmd(t, dir, "feature", "Add OAuth")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Started workflow") {
		t.Errorf("expected 'Started workflow' in output:\n%s", out)
	}

	// State directory should now exist.
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if _, statErr := os.Stat(stateDir); os.IsNotExist(statErr) {
		t.Error("expected state directory to exist after normal start")
	}
}
