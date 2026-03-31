package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCheckpoint_WritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	cp := CheckpointState{
		WorkflowID:    "feature/1-add-auth",
		Step:          3,
		Role:          "coder",
		FilesModified: []string{"src/auth.go", "src/auth_test.go"},
		Notes:         "halfway through token refresh impl",
	}

	if err := SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	p := checkpointPath(dir, cp.WorkflowID)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Errorf("expected checkpoint file at %s, but it does not exist", p)
	}
}

func TestLoadCheckpoint_ReadsBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	cp := CheckpointState{
		WorkflowID:     "bugfix/42-fix-crash",
		Step:           2,
		Role:           "coder",
		PartialHandoff: ".teamwork/handoffs/02-coder-partial.md",
		FilesModified:  []string{"internal/foo/bar.go"},
		Notes:          "almost done",
	}

	if err := SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	got, err := LoadCheckpoint(dir, cp.WorkflowID)
	if err != nil {
		t.Fatalf("LoadCheckpoint: %v", err)
	}
	if got == nil {
		t.Fatal("LoadCheckpoint returned nil, expected checkpoint")
	}

	if got.WorkflowID != cp.WorkflowID {
		t.Errorf("WorkflowID: got %q, want %q", got.WorkflowID, cp.WorkflowID)
	}
	if got.Step != cp.Step {
		t.Errorf("Step: got %d, want %d", got.Step, cp.Step)
	}
	if got.Role != cp.Role {
		t.Errorf("Role: got %q, want %q", got.Role, cp.Role)
	}
	if got.PartialHandoff != cp.PartialHandoff {
		t.Errorf("PartialHandoff: got %q, want %q", got.PartialHandoff, cp.PartialHandoff)
	}
	if len(got.FilesModified) != len(cp.FilesModified) {
		t.Errorf("FilesModified len: got %d, want %d", len(got.FilesModified), len(cp.FilesModified))
	}
	if got.Notes != cp.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, cp.Notes)
	}
	if got.SavedAt == "" {
		t.Error("SavedAt should be set by SaveCheckpoint")
	}
}

func TestLoadCheckpoint_ReturnsNilWhenNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	cp, err := LoadCheckpoint(dir, "nonexistent-workflow")
	if err != nil {
		t.Fatalf("LoadCheckpoint: expected nil error for missing file, got: %v", err)
	}
	if cp != nil {
		t.Errorf("LoadCheckpoint: expected nil for missing file, got: %+v", cp)
	}
}

func TestClearCheckpoint_DeletesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "state"), 0o755); err != nil {
		t.Fatal(err)
	}

	cp := CheckpointState{
		WorkflowID: "feature/99-cleanup",
		Step:       1,
		Role:       "planner",
	}

	if err := SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	p := checkpointPath(dir, cp.WorkflowID)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatal("checkpoint file should exist before clear")
	}

	if err := ClearCheckpoint(dir, cp.WorkflowID); err != nil {
		t.Fatalf("ClearCheckpoint: %v", err)
	}

	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("checkpoint file should not exist after clear")
	}
}

func TestClearCheckpoint_NoopWhenMissing(t *testing.T) {
	dir := t.TempDir()
	// ClearCheckpoint on a workflow that never had a checkpoint should not error.
	if err := ClearCheckpoint(dir, "never-saved"); err != nil {
		t.Errorf("ClearCheckpoint on missing file: expected nil, got: %v", err)
	}
}

func TestCheckpointPath_SanitizesSlashes(t *testing.T) {
	dir := "/tmp/test"
	p := checkpointPath(dir, "feature/1-foo")
	base := filepath.Base(p)
	if base != ".checkpoint-feature-1-foo.yaml" {
		t.Errorf("unexpected checkpoint filename: %q", base)
	}
}
