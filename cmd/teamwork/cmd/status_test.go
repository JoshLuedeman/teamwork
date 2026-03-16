package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// resetStatusFlags resets the status command flags between tests.
func resetStatusFlags(t *testing.T) {
	t.Helper()
	statusCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

// executeStatusCmd runs "teamwork status" with the given args and captures stdout.
func executeStatusCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	resetStatusFlags(t)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"status", "--dir", dir}, args...))

	err := rootCmd.Execute()
	return buf.String(), err
}

// setupStatusWorkflows creates a temp dir with a config and the given workflow
// state files so that `teamwork status` can load them.
func setupStatusWorkflows(t *testing.T, workflows ...*state.WorkflowState) string {
	t.Helper()
	dir := t.TempDir()

	// Write minimal config.
	writeTestConfig(t, dir, minimalConfig)

	// Create state dir and write each workflow.
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, w := range workflows {
		if err := w.Save(dir); err != nil {
			t.Fatalf("saving workflow %s: %v", w.ID, err)
		}
	}

	return dir
}

func TestStatus_NoWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	// Create empty state dir so LoadAll doesn't fail.
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeStatusCmd(t, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No active workflows.") {
		t.Errorf("expected 'No active workflows.' in output:\n%s", out)
	}
}

func TestStatus_TableDefault(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			CurrentStep: 2, CurrentRole: "coder",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
			Steps: []state.StepRecord{},
		},
	)

	out, err := executeStatusCmd(t, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Check header.
	if !strings.Contains(out, "ID") || !strings.Contains(out, "Type") || !strings.Contains(out, "Status") {
		t.Errorf("expected table header in output:\n%s", out)
	}
	// Check workflow row.
	if !strings.Contains(out, "feat-1") {
		t.Errorf("expected 'feat-1' in output:\n%s", out)
	}
	if !strings.Contains(out, "feature") {
		t.Errorf("expected 'feature' in output:\n%s", out)
	}
	if !strings.Contains(out, "active") {
		t.Errorf("expected 'active' in output:\n%s", out)
	}
}

func TestStatus_FilterByStatus(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			CurrentStep: 1, CurrentRole: "planner",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "feat-2", Type: "feature", Status: "completed",
			CurrentStep: 5, CurrentRole: "documenter",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-03T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "bug-1", Type: "bugfix", Status: "blocked",
			CurrentStep: 3, CurrentRole: "tester",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-04T00:00:00Z",
		},
	)

	tests := []struct {
		name       string
		filter     string
		wantIDs    []string
		notWantIDs []string
	}{
		{
			name:       "filter active",
			filter:     "active",
			wantIDs:    []string{"feat-1"},
			notWantIDs: []string{"feat-2", "bug-1"},
		},
		{
			name:       "filter completed",
			filter:     "completed",
			wantIDs:    []string{"feat-2"},
			notWantIDs: []string{"feat-1", "bug-1"},
		},
		{
			name:       "filter blocked",
			filter:     "blocked",
			wantIDs:    []string{"bug-1"},
			notWantIDs: []string{"feat-1", "feat-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := executeStatusCmd(t, dir, "--status", tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, out)
			}
			for _, id := range tt.wantIDs {
				if !strings.Contains(out, id) {
					t.Errorf("expected %q in output:\n%s", id, out)
				}
			}
			for _, id := range tt.notWantIDs {
				if strings.Contains(out, id) {
					t.Errorf("did not expect %q in output:\n%s", id, out)
				}
			}
		})
	}
}

func TestStatus_FilterByType(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			CurrentStep: 1, CurrentRole: "planner",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "bug-1", Type: "bugfix", Status: "active",
			CurrentStep: 2, CurrentRole: "coder",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-03T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--type", "bugfix")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "bug-1") {
		t.Errorf("expected 'bug-1' in output:\n%s", out)
	}
	if strings.Contains(out, "feat-1") {
		t.Errorf("did not expect 'feat-1' in output:\n%s", out)
	}
}

func TestStatus_FilterByStatusAndType(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-active", Type: "feature", Status: "active",
			CurrentStep: 1, CurrentRole: "planner",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "feat-done", Type: "feature", Status: "completed",
			CurrentStep: 5, CurrentRole: "documenter",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-03T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "bug-active", Type: "bugfix", Status: "active",
			CurrentStep: 2, CurrentRole: "coder",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-04T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--status", "active", "--type", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "feat-active") {
		t.Errorf("expected 'feat-active' in output:\n%s", out)
	}
	if strings.Contains(out, "feat-done") {
		t.Errorf("did not expect 'feat-done' in output:\n%s", out)
	}
	if strings.Contains(out, "bug-active") {
		t.Errorf("did not expect 'bug-active' in output:\n%s", out)
	}
}

func TestStatus_FilterNoResults(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			CurrentStep: 1, CurrentRole: "planner",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--status", "failed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No active workflows.") {
		t.Errorf("expected 'No active workflows.' in output:\n%s", out)
	}
}

func TestStatus_JSONFormat(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			Goal:        "Add auth",
			CurrentStep: 2, CurrentRole: "coder",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "bug-1", Type: "bugfix", Status: "blocked",
			Goal:        "Fix crash",
			CurrentStep: 1, CurrentRole: "tester",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-03T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result []state.WorkflowState
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse error: %v\noutput: %s", err, out)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 workflows in JSON, got %d", len(result))
	}

	// Verify fields are correctly serialized.
	found := false
	for _, w := range result {
		if w.ID == "feat-1" {
			found = true
			if w.Type != "feature" {
				t.Errorf("expected type 'feature', got %q", w.Type)
			}
			if w.Status != "active" {
				t.Errorf("expected status 'active', got %q", w.Status)
			}
			if w.Goal != "Add auth" {
				t.Errorf("expected goal 'Add auth', got %q", w.Goal)
			}
			if w.CurrentStep != 2 {
				t.Errorf("expected current_step 2, got %d", w.CurrentStep)
			}
			if w.CurrentRole != "coder" {
				t.Errorf("expected current_role 'coder', got %q", w.CurrentRole)
			}
		}
	}
	if !found {
		t.Error("expected to find workflow 'feat-1' in JSON output")
	}
}

func TestStatus_JSONFormatEmpty(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeStatusCmd(t, dir, "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result []state.WorkflowState
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse error: %v\noutput: %s", err, out)
	}
	if len(result) != 0 {
		t.Errorf("expected empty JSON array, got %d items", len(result))
	}
}

func TestStatus_JSONWithFilter(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			CurrentStep: 1, CurrentRole: "planner",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
		&state.WorkflowState{
			ID: "bug-1", Type: "bugfix", Status: "blocked",
			CurrentStep: 3, CurrentRole: "tester",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-03T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--format", "json", "--status", "blocked")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result []state.WorkflowState
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("JSON parse error: %v\noutput: %s", err, out)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 workflow in JSON, got %d", len(result))
	}
	if result[0].ID != "bug-1" {
		t.Errorf("expected ID 'bug-1', got %q", result[0].ID)
	}
}

func TestStatus_YAMLFormat(t *testing.T) {
	dir := setupStatusWorkflows(t,
		&state.WorkflowState{
			ID: "feat-1", Type: "feature", Status: "active",
			Goal:        "Add auth",
			CurrentStep: 2, CurrentRole: "coder",
			CreatedAt: "2025-01-01T00:00:00Z", UpdatedAt: "2025-01-02T00:00:00Z",
		},
	)

	out, err := executeStatusCmd(t, dir, "--format", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result []state.WorkflowState
	if err := yaml.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("YAML parse error: %v\noutput: %s", err, out)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 workflow in YAML, got %d", len(result))
	}
	if result[0].ID != "feat-1" {
		t.Errorf("expected ID 'feat-1', got %q", result[0].ID)
	}
	if result[0].Goal != "Add auth" {
		t.Errorf("expected goal 'Add auth', got %q", result[0].Goal)
	}
}

func TestStatus_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	_, err := executeStatusCmd(t, dir, "--format", "csv")
	if err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected 'unknown format' error, got: %v", err)
	}
}

func TestStatus_InvalidStatus(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	_, err := executeStatusCmd(t, dir, "--status", "running")
	if err == nil {
		t.Fatal("expected error for unknown status, got nil")
	}
	if !strings.Contains(err.Error(), "unknown status") {
		t.Errorf("expected 'unknown status' error, got: %v", err)
	}
}
