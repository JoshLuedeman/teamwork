package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/pflag"
)

// resetTimelineFlags resets timeline command flags between tests.
func resetTimelineFlags(t *testing.T) {
	t.Helper()
	timelineCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeTimelineCmd runs "teamwork timeline" with the given args and captures stdout.
func executeTimelineCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	resetTimelineFlags(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"timeline", "--dir", dir}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

// setupTimelineWorkflow creates a temp dir with a config and a workflow state
// containing completed, active, and pending steps.
func setupTimelineWorkflow(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()

	// Write minimal config.
	writeTestConfig(t, dir, minimalConfig)

	// Create state dir.
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Build a workflow with 3 steps: step 1 completed, step 2 active, step 3 pending.
	ws := &state.WorkflowState{
		ID:          "feature/1-auth",
		Type:        "feature",
		Status:      "active",
		CurrentStep: 2,
		CurrentRole: "coder",
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-02T00:00:00Z",
		Steps: []state.StepRecord{
			{
				Step:      1,
				Role:      "planner",
				Action:    "Plan implementation",
				Started:   "2025-01-01T00:00:00Z",
				Completed: "2025-01-01T01:00:00Z",
				Handoff:   "01-planner.md",
			},
			{
				Step:    2,
				Role:    "coder",
				Action:  "Implement feature",
				Started: "2025-01-01T01:00:00Z",
			},
			{
				Step:    3,
				Role:    "tester",
				Action:  "Write and run tests",
				Started: "",
			},
		},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatalf("saving workflow: %v", err)
	}
	return dir, ws.ID
}

func TestTimeline_ASCIITable(t *testing.T) {
	dir, workflowID := setupTimelineWorkflow(t)

	out, err := executeTimelineCmd(t, dir, workflowID)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Header and workflow info.
	if !strings.Contains(out, workflowID) {
		t.Errorf("expected workflow ID in output:\n%s", out)
	}
	if !strings.Contains(out, "Step") {
		t.Errorf("expected 'Step' column header in output:\n%s", out)
	}
	if !strings.Contains(out, "Role") {
		t.Errorf("expected 'Role' column header in output:\n%s", out)
	}
	if !strings.Contains(out, "Status") {
		t.Errorf("expected 'Status' column header in output:\n%s", out)
	}

	// Rows for each step role.
	if !strings.Contains(out, "planner") {
		t.Errorf("expected 'planner' row in output:\n%s", out)
	}
	if !strings.Contains(out, "coder") {
		t.Errorf("expected 'coder' row in output:\n%s", out)
	}
	if !strings.Contains(out, "tester") {
		t.Errorf("expected 'tester' row in output:\n%s", out)
	}

	// Status symbols.
	if !strings.Contains(out, "completed") {
		t.Errorf("expected 'completed' status in output:\n%s", out)
	}
	if !strings.Contains(out, "active") {
		t.Errorf("expected 'active' status in output:\n%s", out)
	}
	if !strings.Contains(out, "pending") {
		t.Errorf("expected 'pending' status in output:\n%s", out)
	}

	// Handoff file for completed step.
	if !strings.Contains(out, "01-planner.md") {
		t.Errorf("expected handoff file in output:\n%s", out)
	}
}

func TestTimeline_MermaidFlag(t *testing.T) {
	dir, workflowID := setupTimelineWorkflow(t)

	out, err := executeTimelineCmd(t, dir, workflowID, "--mermaid")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.HasPrefix(strings.TrimSpace(out), "gantt") {
		t.Errorf("expected mermaid output to start with 'gantt', got:\n%s", out)
	}
	if !strings.Contains(out, "dateFormat") {
		t.Errorf("expected 'dateFormat' directive in mermaid output:\n%s", out)
	}
	if !strings.Contains(out, "planner") {
		t.Errorf("expected 'planner' task in mermaid output:\n%s", out)
	}
}

func TestTimeline_MissingWorkflowID(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"timeline", "--dir", dir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no workflow ID provided, got nil")
	}
}

func TestTimeline_UnknownWorkflowID(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := executeTimelineCmd(t, dir, "nonexistent-workflow")
	if err == nil {
		t.Fatal("expected error for unknown workflow ID, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-workflow") {
		t.Errorf("expected error to mention workflow ID, got: %v", err)
	}
}
