package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/state"
)

func executeHandoffInitCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"handoff", "init", "--dir", dir}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func writeWorkflowState(t *testing.T, dir, workflowID string, ws *state.WorkflowState) {
	t.Helper()
	if err := ws.Save(dir); err != nil {
		t.Fatalf("saving workflow state: %v", err)
	}
}

func TestHandoffInit_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	// Create a workflow state at step 4 (coder).
	ws := state.New("feature/1-test", "feature", "Test goal")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	writeWorkflowState(t, dir, "feature/1-test", ws)

	out, err := executeHandoffInitCmd(t, dir, "feature/1-test")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Created:") {
		t.Errorf("expected 'Created:' in output, got: %s", out)
	}

	// Verify the file was created with template content.
	p := filepath.Join(dir, ".teamwork", "handoffs", "feature/1-test", "step-04-coder.md")
	data, readErr := os.ReadFile(p)
	if readErr != nil {
		t.Fatalf("expected handoff file to exist at %s: %v", p, readErr)
	}

	content := string(data)
	if !strings.Contains(content, "## Files Changed") {
		t.Errorf("expected '## Files Changed' in handoff content, got:\n%s", content)
	}
	if !strings.Contains(content, "## How to Test") {
		t.Errorf("expected '## How to Test' in handoff content, got:\n%s", content)
	}
}

func TestHandoffInit_SkipsIfFileExists(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	ws := state.New("feature/2-test", "feature", "Another goal")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement", Started: "2024-01-01T00:00:00Z"},
	}
	writeWorkflowState(t, dir, "feature/2-test", ws)

	// Pre-create the handoff file.
	p := filepath.Join(dir, ".teamwork", "handoffs", "feature/2-test", "step-04-coder.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("existing content"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := executeHandoffInitCmd(t, dir, "feature/2-test")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Warning") {
		t.Errorf("expected warning message, got: %s", out)
	}

	// File should still have original content.
	data, _ := os.ReadFile(p)
	if string(data) != "existing content" {
		t.Errorf("expected file content to be unchanged, got: %s", string(data))
	}
}
