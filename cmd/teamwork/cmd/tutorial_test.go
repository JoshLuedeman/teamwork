package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// resetTutorialFlags resets the tutorial command flags between tests to avoid
// pollution from previous test runs.
func resetTutorialFlags(t *testing.T) {
	t.Helper()
	tutorialCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeTutorialCmd runs "teamwork tutorial" with the given args and captures stdout.
func executeTutorialCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetTutorialFlags(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"tutorial"}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestTutorial_NonInteractive_ContainsHeader(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Teamwork Tutorial: Your First Feature Workflow") {
		t.Errorf("expected tutorial header in output:\n%s", out)
	}
	if !strings.Contains(out, "No files will be created") {
		t.Errorf("expected dry-run notice in output:\n%s", out)
	}
}

func TestTutorial_NonInteractive_ContainsAllSteps(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	steps := []string{
		"Step 1",
		"Step 2",
		"Step 3",
		"Step 4",
		"Step 5",
	}
	for _, step := range steps {
		if !strings.Contains(out, step) {
			t.Errorf("expected %q in output:\n%s", step, out)
		}
	}
}

func TestTutorial_NonInteractive_ContainsRoles(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	roles := []string{
		"Human",
		"Planner",
		"Architect",
		"Coder",
		"Tester",
		"Reviewer",
		"Documenter",
	}
	for _, role := range roles {
		if !strings.Contains(out, role) {
			t.Errorf("expected role %q in output:\n%s", role, out)
		}
	}
}

func TestTutorial_NonInteractive_ContainsStateTransitions(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	transitions := []string{
		"State transition",
		"CREATED",
		"IN_PROGRESS",
		"COMPLETED",
	}
	for _, t2 := range transitions {
		if !strings.Contains(out, t2) {
			t.Errorf("expected %q in output:\n%s", t2, out)
		}
	}
}

func TestTutorial_NonInteractive_ContainsExampleCommands(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	commands := []string{
		"teamwork start feature",
		"teamwork approve",
		"teamwork status",
		"teamwork history",
	}
	for _, cmd := range commands {
		if !strings.Contains(out, cmd) {
			t.Errorf("expected command example %q in output:\n%s", cmd, out)
		}
	}
}

func TestTutorial_NonInteractive_ContainsQualityGates(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	gates := []string{
		"handoff_complete",
		"tests_pass",
		"lint_pass",
	}
	for _, gate := range gates {
		if !strings.Contains(out, gate) {
			t.Errorf("expected quality gate %q in output:\n%s", gate, out)
		}
	}
}

func TestTutorial_NonInteractive_ContainsFooter(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Tutorial Complete") {
		t.Errorf("expected footer in output:\n%s", out)
	}
	if !strings.Contains(out, "Ready to start for real") {
		t.Errorf("expected call-to-action in output:\n%s", out)
	}
}

func TestTutorial_NonInteractive_ContainsSeparators(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// There should be separators between steps (4 separators for 5 steps).
	separatorCount := strings.Count(out, strings.Repeat("─", 60))
	if separatorCount != 4 {
		t.Errorf("expected 4 separators between 5 steps, got %d", separatorCount)
	}
}

func TestTutorial_NonInteractive_NoPressEnterPrompt(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if strings.Contains(out, "Press Enter") {
		t.Errorf("non-interactive mode should not contain 'Press Enter' prompt:\n%s", out)
	}
}

func TestTutorial_NonInteractive_HandoffArtifactExample(t *testing.T) {
	out, err := executeTutorialCmd(t, "--non-interactive")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Handoff artifact") {
		t.Errorf("expected handoff artifact example in output:\n%s", out)
	}
}

func TestTutorial_StepCount(t *testing.T) {
	steps := tutorialSteps()
	if len(steps) != 5 {
		t.Errorf("expected 5 tutorial steps, got %d", len(steps))
	}
}
