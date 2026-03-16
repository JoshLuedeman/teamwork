package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshluedeman/teamwork/internal/config"
)

func writeCustomConfig(t *testing.T, dir string) {
	t.Helper()
	twDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(twDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "model:\n  provider: \"openai\"\n  name: \"gpt-4\"\ncustom_workflows:\n  data-pipeline:\n    steps:\n      - role: \"planner\"\n        description: \"Design the data pipeline\"\n      - role: \"coder\"\n        description: \"Implement the pipeline\"\n      - role: \"tester\"\n        description: \"Verify pipeline output\"\n"
	if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStartCustomWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeCustomConfig(t, dir)
	eng, err := NewEngine(dir)
	if err != nil { t.Fatal(err) }
	ws, err := eng.Start("data-pipeline", "build pipeline", 42)
	if err != nil { t.Fatalf("Start returned error: %v", err) }
	if ws.Type != "data-pipeline" { t.Errorf("Type = %q, want data-pipeline", ws.Type) }
	if ws.CurrentRole != "planner" { t.Errorf("CurrentRole = %q, want planner", ws.CurrentRole) }
}

func TestStartUnknownType(t *testing.T) {
	dir := t.TempDir()
	writeCustomConfig(t, dir)
	eng, err := NewEngine(dir)
	if err != nil { t.Fatal(err) }
	_, err = eng.Start("does-not-exist", "goal", 1)
	if err == nil { t.Fatal("expected error for unknown type") }
}

func TestBuiltinStillWorks(t *testing.T) {
	dir := t.TempDir()
	writeCustomConfig(t, dir)
	eng, err := NewEngine(dir)
	if err != nil { t.Fatal(err) }
	ws, err := eng.Start("feature", "new feature", 10)
	if err != nil { t.Fatalf("Start returned error: %v", err) }
	if ws.Type != "feature" { t.Errorf("Type = %q, want feature", ws.Type) }
}

func TestCustomDefinition(t *testing.T) {
	cfg := &config.Config{
		CustomWorkflows: map[string]config.CustomWorkflow{
			"my-flow": {
				Steps: []config.CustomStep{
					{Role: "planner", Description: "plan it"},
					{Role: "coder", Description: "code it"},
				},
			},
		},
	}
	def, ok := CustomDefinition(cfg, "my-flow")
	if !ok { t.Fatal("expected to find custom definition") }
	if len(def.Steps) != 2 { t.Fatalf("Steps = %d, want 2", len(def.Steps)) }
	if def.Steps[0].Role != "planner" { t.Errorf("Step 0 Role = %q, want planner", def.Steps[0].Role) }
	_, ok = CustomDefinition(cfg, "missing")
	if ok { t.Error("expected not found for missing type") }
	_, ok = CustomDefinition(nil, "anything")
	if ok { t.Error("expected not found for nil config") }
}

func TestIsBuiltinType(t *testing.T) {
	if !IsBuiltinType("feature") { t.Error("feature should be builtin") }
	if IsBuiltinType("my-custom") { t.Error("my-custom should not be builtin") }
}

func TestPreviewStepsWithConfig(t *testing.T) {
	cfg := &config.Config{
		CustomWorkflows: map[string]config.CustomWorkflow{
			"my-flow": {
				Steps: []config.CustomStep{
					{Role: "planner", Description: "plan it"},
					{Role: "coder", Description: "code it"},
				},
			},
		},
	}
	steps, err := PreviewStepsWithConfig(cfg, "my-flow")
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if len(steps) != 2 { t.Fatalf("Steps = %d, want 2", len(steps)) }
	steps, err = PreviewStepsWithConfig(cfg, "feature")
	if err != nil { t.Fatalf("unexpected error for builtin: %v", err) }
	if len(steps) == 0 { t.Error("expected steps for builtin type") }
	_, err = PreviewStepsWithConfig(cfg, "nonexistent")
	if err == nil { t.Error("expected error for unknown type") }
}
