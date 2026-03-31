package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// buildFixture writes minimal .teamwork/ state and handoff fixture files to dir
// and returns the workflow ID used.
func buildFixture(t *testing.T, dir string) string {
	t.Helper()

	workflowID := "feature/1-add-version-flag"

	// Write state file.
	stateDir := filepath.Join(dir, ".teamwork", "state", "feature")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}

	type stepRecord struct {
		Step        int    `yaml:"step"`
		Role        string `yaml:"role"`
		Action      string `yaml:"action"`
		Started     string `yaml:"started"`
		Completed   string `yaml:"completed,omitempty"`
		Handoff     string `yaml:"handoff,omitempty"`
		QualityGate string `yaml:"quality_gate,omitempty"`
	}
	type workflowState struct {
		ID          string       `yaml:"id"`
		Type        string       `yaml:"type"`
		Status      string       `yaml:"status"`
		Goal        string       `yaml:"goal"`
		CurrentStep int          `yaml:"current_step"`
		CurrentRole string       `yaml:"current_role"`
		Steps       []stepRecord `yaml:"steps"`
		CreatedAt   string       `yaml:"created_at"`
		UpdatedAt   string       `yaml:"updated_at"`
		CreatedBy   string       `yaml:"created_by"`
	}

	ws := workflowState{
		ID:          workflowID,
		Type:        "feature",
		Status:      "completed",
		Goal:        "Add --version flag to CLI",
		CurrentStep: 3,
		CurrentRole: "reviewer",
		Steps: []stepRecord{
			{
				Step:        1,
				Role:        "planner",
				Action:      "plan",
				Started:     "2025-01-10T10:00:00Z",
				Completed:   "2025-01-10T10:15:00Z",
				QualityGate: "passed",
			},
			{
				Step:        2,
				Role:        "coder",
				Action:      "implement",
				Started:     "2025-01-10T10:20:00Z",
				Completed:   "2025-01-10T10:55:00Z",
				QualityGate: "passed",
			},
			{
				Step:      3,
				Role:      "reviewer",
				Action:    "review",
				Started:   "2025-01-10T11:00:00Z",
				Completed: "2025-01-10T11:20:00Z",
			},
		},
		CreatedAt: "2025-01-10T10:00:00Z",
		UpdatedAt: "2025-01-10T11:20:00Z",
		CreatedBy: "human",
	}
	data, err := yaml.Marshal(ws)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	stateFile := filepath.Join(stateDir, "1-add-version-flag.yaml")
	if err := os.WriteFile(stateFile, data, 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	// Write a handoff artifact for step 2.
	handoffDir := filepath.Join(dir, ".teamwork", "handoffs", workflowID)
	if err := os.MkdirAll(handoffDir, 0o755); err != nil {
		t.Fatalf("mkdir handoff: %v", err)
	}
	handoffContent := `# Handoff: coder → reviewer

## Summary

Implemented the --version flag by reading version from ldflags. All tests pass.

## Artifacts

- cmd/main.go
`
	if err := os.WriteFile(filepath.Join(handoffDir, "02-coder.md"), []byte(handoffContent), 0o644); err != nil {
		t.Fatalf("write handoff: %v", err)
	}

	return workflowID
}

func TestBuild_CorrectStepCountAndGoal(t *testing.T) {
	dir := t.TempDir()
	workflowID := buildFixture(t, dir)

	r, err := Build(dir, workflowID)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if r.Goal != "Add --version flag to CLI" {
		t.Errorf("Goal = %q, want %q", r.Goal, "Add --version flag to CLI")
	}
	if len(r.Steps) != 3 {
		t.Errorf("len(Steps) = %d, want 3", len(r.Steps))
	}
	if r.WorkflowID != workflowID {
		t.Errorf("WorkflowID = %q, want %q", r.WorkflowID, workflowID)
	}
	if r.Status != "completed" {
		t.Errorf("Status = %q, want completed", r.Status)
	}
}

func TestBuild_GatePassRate(t *testing.T) {
	dir := t.TempDir()
	workflowID := buildFixture(t, dir)

	r, err := Build(dir, workflowID)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// 2 gates in fixture (steps 1 and 2), both passed → 100%.
	if r.GatePassRate != 1.0 {
		t.Errorf("GatePassRate = %v, want 1.0", r.GatePassRate)
	}
}

func TestRenderMarkdown_ContainsWorkflowIDAndGoal(t *testing.T) {
	dir := t.TempDir()
	workflowID := buildFixture(t, dir)

	r, err := Build(dir, workflowID)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	md := RenderMarkdown(r)

	if !strings.Contains(md, workflowID) {
		t.Errorf("Markdown does not contain workflow ID %q", workflowID)
	}
	if !strings.Contains(md, r.Goal) {
		t.Errorf("Markdown does not contain goal %q", r.Goal)
	}
}

func TestRenderJSON_ValidJSONWithWorkflowIDKey(t *testing.T) {
	dir := t.TempDir()
	workflowID := buildFixture(t, dir)

	r, err := Build(dir, workflowID)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	data, err := RenderJSON(r)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := m["workflow_id"]; !ok {
		t.Errorf("JSON missing 'workflow_id' key; got keys: %v", keys(m))
	}
}

func TestRenderHTML_ContainsWorkflowID(t *testing.T) {
	r := &Report{
		WorkflowID: "test/my-workflow",
		Goal:       "Test the HTML renderer",
		Status:     "completed",
	}
	html := RenderHTML(r)
	if !strings.Contains(html, "test/my-workflow") {
		t.Errorf("HTML does not contain workflow ID")
	}
}

func keys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
