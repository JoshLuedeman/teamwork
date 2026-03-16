package workflow

import (
	"os"
	"path/filepath"
	"strings"
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
		GateResults: map[string]bool{
			"tests_pass": true,
			"lint_pass":  true,
		},
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

func TestCompleteRecordsNonZeroDuration(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"state/feature", "handoffs", "metrics"} {
		if err := os.MkdirAll(filepath.Join(dir, ".teamwork", sub), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Feature workflow has 9 steps; place the workflow on step 9 (the final step).
	ws := state.New("feature/1-complete-test", "feature", "Test goal")
	ws.CurrentStep = 9
	ws.CurrentRole = "documenter"
	ws.Steps = []state.StepRecord{
		{
			Step:    9,
			Role:    "documenter",
			Action:  "Update docs and changelog",
			Started: time.Now().UTC().Add(-10 * time.Second).Format(time.RFC3339),
		},
	}
	if err := ws.Save(dir); err != nil {
		t.Fatalf("save state: %v", err)
	}

	cfg := config.Default()
	eng := &Engine{Dir: dir, Config: cfg}

	if err := eng.Complete("feature/1-complete-test"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	events, err := metrics.Load(dir, "feature/1-complete-test")
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
		t.Error("Complete() should record non-zero duration_sec; got 0 — duration must be calculated before ws.Complete() marks the step done")
	}
	if completeDuration < 9 {
		t.Errorf("duration_sec = %d, expected >= 9 (step was backdated by 10s)", completeDuration)
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

// setupEngine creates a temp directory with the required .teamwork subdirectories,
// an active workflow state file, and returns an Engine ready for testing.
func setupEngine(t *testing.T, gates config.QualityGatesConfig) (*Engine, string) {
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
	cfg.QualityGates = gates

	return &Engine{Dir: dir, Config: cfg}, dir
}

// validArtifact returns a minimal valid handoff artifact for testing.
func validArtifact(gateResults map[string]bool) *handoff.Artifact {
	return &handoff.Artifact{
		WorkflowID:  "feature/1-test",
		Step:        4,
		Role:        "coder",
		NextRole:    "tester",
		Date:        "2025-01-01T00:00:00Z",
		Summary:     "Implemented feature",
		Context:     "Ready for testing",
		GateResults: gateResults,
	}
}

func TestHandoff_QualityGates(t *testing.T) {
	tests := []struct {
		name        string
		gates       config.QualityGatesConfig
		gateResults map[string]bool
		wantErr     string // substring expected in error; empty means no error
	}{
		{
			name: "all gates pass",
			gates: config.QualityGatesConfig{
				TestsPass: true,
				LintPass:  true,
			},
			gateResults: map[string]bool{
				"tests_pass": true,
				"lint_pass":  true,
			},
			wantErr: "",
		},
		{
			name: "tests_pass gate fails",
			gates: config.QualityGatesConfig{
				TestsPass: true,
				LintPass:  false,
			},
			gateResults: map[string]bool{
				"tests_pass": false,
			},
			wantErr: `quality gate "tests_pass" failed`,
		},
		{
			name: "lint_pass gate fails",
			gates: config.QualityGatesConfig{
				TestsPass: false,
				LintPass:  true,
			},
			gateResults: map[string]bool{
				"lint_pass": false,
			},
			wantErr: `quality gate "lint_pass" failed`,
		},
		{
			name: "tests_pass gate not reported",
			gates: config.QualityGatesConfig{
				TestsPass: true,
				LintPass:  false,
			},
			gateResults: map[string]bool{},
			wantErr:     `quality gate "tests_pass" required but not reported`,
		},
		{
			name: "gates disabled allows handoff",
			gates: config.QualityGatesConfig{
				TestsPass: false,
				LintPass:  false,
			},
			gateResults: nil,
			wantErr:     "",
		},
		{
			name: "only tests_pass enabled and passes",
			gates: config.QualityGatesConfig{
				TestsPass: true,
				LintPass:  false,
			},
			gateResults: map[string]bool{
				"tests_pass": true,
			},
			wantErr: "",
		},
		{
			name: "only lint_pass enabled and passes",
			gates: config.QualityGatesConfig{
				TestsPass: false,
				LintPass:  true,
			},
			gateResults: map[string]bool{
				"lint_pass": true,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng, _ := setupEngine(t, tt.gates)
			artifact := validArtifact(tt.gateResults)

			err := eng.Handoff("feature/1-test", artifact)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Handoff() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Handoff() expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Handoff() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
