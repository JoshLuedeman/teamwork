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

// resetResumeFlags resets resume command flags between tests.
func resetResumeFlags(t *testing.T) {
	t.Helper()
	resumeCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeResumeCmd runs "teamwork resume" with the given args and captures stdout.
func executeResumeCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	resetResumeFlags(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"resume", "--dir", dir}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

// setupResumeDir creates a minimal temp dir with config and state directory.
func setupResumeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResume_CheckpointExists(t *testing.T) {
	dir := setupResumeDir(t)

	cp := state.CheckpointState{
		WorkflowID:    "feature/1-auth",
		Step:          3,
		Role:          "coder",
		FilesModified: []string{"src/auth.go", "src/auth_test.go"},
		Notes:         "halfway through implementation",
	}
	if err := state.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	out, err := executeResumeCmd(t, dir, "feature/1-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "feature/1-auth") {
		t.Errorf("expected workflow ID in output:\n%s", out)
	}
	if !strings.Contains(out, "3") {
		t.Errorf("expected step number in output:\n%s", out)
	}
	if !strings.Contains(out, "coder") {
		t.Errorf("expected role in output:\n%s", out)
	}
	if !strings.Contains(out, "src/auth.go") {
		t.Errorf("expected modified file in output:\n%s", out)
	}
	if !strings.Contains(out, "halfway through implementation") {
		t.Errorf("expected notes in output:\n%s", out)
	}
}

func TestResume_NoCheckpoint(t *testing.T) {
	dir := setupResumeDir(t)

	out, err := executeResumeCmd(t, dir, "nonexistent-workflow")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No checkpoint found") {
		t.Errorf("expected 'No checkpoint found' message, got:\n%s", out)
	}
	if !strings.Contains(out, "nonexistent-workflow") {
		t.Errorf("expected workflow ID in message, got:\n%s", out)
	}
}

func TestResume_ClearFlag(t *testing.T) {
	dir := setupResumeDir(t)

	cp := state.CheckpointState{
		WorkflowID: "feature/99-clear-test",
		Step:       1,
		Role:       "planner",
	}
	if err := state.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	// Verify checkpoint exists before clearing.
	loaded, err := state.LoadCheckpoint(dir, cp.WorkflowID)
	if err != nil || loaded == nil {
		t.Fatal("checkpoint should exist before --clear")
	}

	out, err := executeResumeCmd(t, dir, "feature/99-clear-test", "--clear")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "cleared") {
		t.Errorf("expected 'cleared' confirmation in output:\n%s", out)
	}

	// Checkpoint should be gone.
	after, err := state.LoadCheckpoint(dir, cp.WorkflowID)
	if err != nil {
		t.Fatalf("LoadCheckpoint after clear: %v", err)
	}
	if after != nil {
		t.Error("checkpoint should not exist after --clear")
	}
}
