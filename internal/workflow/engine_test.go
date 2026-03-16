package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/joshluedeman/teamwork/internal/handoff"
	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/joshluedeman/teamwork/internal/state"
)

// setupTestEngine creates a minimal Engine with a temp directory and a
// default config file.
func setupTestEngine(t *testing.T) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()

	// Create the minimal config required by NewEngine.
	cfgDir := filepath.Join(dir, ".teamwork")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.yaml")
	cfgData := []byte("hub_repo: test/repo\n")
	if err := os.WriteFile(cfgFile, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(dir)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return eng, dir
}

// setupEngineWithWorkflow creates a temp directory with an active workflow
// state at step 4 (coder), ready for handoff testing.
func setupEngineWithWorkflow(t *testing.T) (*Engine, string) {
	t.Helper()

	dir := t.TempDir()
	for _, sub := range []string{"state/feature", "handoffs", "metrics"} {
		if err := os.MkdirAll(filepath.Join(dir, ".teamwork", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	ws := state.New("feature/1-test", "feature", "Test goal")
	ws.CurrentStep = 4
	ws.CurrentRole = "coder"
	ws.Steps = []state.StepRecord{
		{Step: 4, Role: "coder", Action: "Implement and open PR", Started: "2025-01-01T00:00:00Z"},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatalf("save state: %v", err)
	}

	cfg := config.Default()
	return &Engine{Dir: dir, Config: cfg}, dir
}

func TestStartCreatesStepRecord(t *testing.T) {
	eng, _ := setupTestEngine(t)

	ws, err := eng.Start("spike", "Investigate caching", 10)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if len(ws.Steps) == 0 {
		t.Fatal("Start should create a StepRecord for step 1")
	}
	if ws.Steps[0].Step != 1 {
		t.Errorf("first StepRecord.Step = %d, want 1", ws.Steps[0].Step)
	}
	if ws.Steps[0].Started == "" {
		t.Error("first StepRecord.Started is empty")
	}

	startedAt, err := ws.CurrentStepStartedAt()
	if err != nil {
		t.Fatalf("CurrentStepStartedAt: %v", err)
	}
	if time.Since(startedAt) > 5*time.Second {
		t.Errorf("step start time is too old: %v", startedAt)
	}
}

func TestHandoffRecordsNonZeroDuration(t *testing.T) {
	eng, dir := setupEngineWithWorkflow(t)

	// Backdate the step start time so elapsed duration > 0.
	ws, err := state.Load(dir, "feature/1-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ws.Steps[0].Started = time.Now().UTC().Add(-10 * time.Second).Format(time.RFC3339)
	if err := ws.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	artifact := &handoff.Artifact{
		WorkflowID: "feature/1-test",
		Step:       4,
		Role:       "coder",
		NextRole:   "tester",
		Date:       "2025-01-01T00:00:00Z",
		Summary:    "Implemented feature",
		Context:    "Ready for testing",
	}

	if err := eng.Handoff("feature/1-test", artifact); err != nil {
		t.Fatalf("Handoff: %v", err)
	}

	events, err := metrics.Load(dir, "feature/1-test")
	if err != nil {
		t.Fatalf("metrics.Load: %v", err)
	}

	var completeDuration int
	for _, ev := range events {
		if ev.Action == metrics.ActionComplete {
			completeDuration = ev.DurationSec
			break
		}
	}

	if completeDuration == 0 {
		t.Error("expected non-zero duration_sec on complete event, got 0")
	}
	if completeDuration < 9 {
		t.Errorf("duration_sec = %d, expected >= 9 (backdated by 10s)", completeDuration)
	}
}

func TestSummarizeAggregatesRealDurations(t *testing.T) {
	events := []metrics.Event{
		{Workflow: "test/1", Step: 1, Role: "planner", Action: metrics.ActionStart},
		{Workflow: "test/1", Step: 1, Role: "planner", Action: metrics.ActionComplete, DurationSec: 120},
		{Workflow: "test/1", Step: 2, Role: "coder", Action: metrics.ActionStart},
		{Workflow: "test/1", Step: 2, Role: "coder", Action: metrics.ActionComplete, DurationSec: 300},
		{Workflow: "test/1", Step: 3, Role: "planner", Action: metrics.ActionStart},
		{Workflow: "test/1", Step: 3, Role: "planner", Action: metrics.ActionComplete, DurationSec: 60},
	}

	s := metrics.Summarize(events)

	if s.TotalDuration != 480 {
		t.Errorf("TotalDuration = %d, want 480", s.TotalDuration)
	}
	if s.RoleDurations["planner"] != 180 {
		t.Errorf("RoleDurations[planner] = %d, want 180", s.RoleDurations["planner"])
	}
	if s.RoleDurations["coder"] != 300 {
		t.Errorf("RoleDurations[coder] = %d, want 300", s.RoleDurations["coder"])
	}
	if s.StepCount != 3 {
		t.Errorf("StepCount = %d, want 3", s.StepCount)
	}
}
