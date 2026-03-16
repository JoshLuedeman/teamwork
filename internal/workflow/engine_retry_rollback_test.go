package workflow

import (
	"testing"
	"time"

	"github.com/joshluedeman/teamwork/internal/state"
)

// createWorkflowInState creates a workflow state file with the given status.
func createWorkflowInState(t *testing.T, dir, wfID, wfType, status string, step int, role string) {
	t.Helper()
	ts := time.Now().UTC().Format(time.RFC3339)
	ws := &state.WorkflowState{
		ID:          wfID,
		Type:        wfType,
		Status:      status,
		CurrentStep: step,
		CurrentRole: role,
		Goal:        "test goal",
		Branch:      wfID,
		CreatedAt:   ts,
		UpdatedAt:   ts,
		Steps:       []state.StepRecord{{Step: step, Role: role, Started: ts}},
	}
	if status == state.StatusFailed {
		ws.Steps[0].Completed = ts
		ws.Steps[0].QualityGate = "failed"
		ws.Blockers = []state.Blocker{{Reason: "test failure", RaisedBy: role, RaisedAt: ts}}
	}
	if status == state.StatusBlocked {
		ws.Blockers = []state.Blocker{{Reason: "blocked reason", RaisedBy: "human", RaisedAt: ts}}
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}
}

// createMultiStepWorkflow creates a workflow at a given step with step records for all prior steps.
func createMultiStepWorkflow(t *testing.T, dir, wfID, wfType, status string, currentStep int) {
	t.Helper()
	ts := time.Now().UTC().Format(time.RFC3339)

	var steps []state.StepRecord
	for s := 1; s < currentStep; s++ {
		steps = append(steps, state.StepRecord{
			Step:      s,
			Role:      "test-role",
			Action:    "test action",
			Started:   ts,
			Completed: ts,
		})
	}
	// Current step (not completed).
	steps = append(steps, state.StepRecord{
		Step:    currentStep,
		Role:    "coder",
		Action:  "current action",
		Started: ts,
	})

	ws := &state.WorkflowState{
		ID:          wfID,
		Type:        wfType,
		Status:      status,
		CurrentStep: currentStep,
		CurrentRole: "coder",
		Goal:        "test goal",
		Branch:      wfID,
		CreatedAt:   ts,
		UpdatedAt:   ts,
		Steps:       steps,
	}
	if status == state.StatusFailed {
		ws.Steps[len(ws.Steps)-1].Completed = ts
		ws.Steps[len(ws.Steps)-1].QualityGate = "failed"
		ws.Blockers = []state.Blocker{{Reason: "test failure", RaisedBy: "coder", RaisedAt: ts}}
	}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}
}

// --- Retry tests ---

func TestRetryFromFailed(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/42-test", "feature", state.StatusFailed, 3, "architect")

	if err := e.Retry("feature/42-test"); err != nil {
		t.Fatalf("Retry failed: %v", err)
	}

	ws, err := state.Load(dir, "feature/42-test")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Status != state.StatusActive {
		t.Errorf("expected status active, got %s", ws.Status)
	}
	if ws.CurrentStep != 3 {
		t.Errorf("expected step 3, got %d", ws.CurrentStep)
	}
	if len(ws.Blockers) != 0 {
		t.Errorf("expected no blockers, got %d", len(ws.Blockers))
	}
	// Step record should have cleared completed and quality_gate.
	for _, sr := range ws.Steps {
		if sr.Step == 3 {
			if sr.Completed != "" {
				t.Errorf("expected cleared completed, got %q", sr.Completed)
			}
			if sr.QualityGate != "" {
				t.Errorf("expected cleared quality_gate, got %q", sr.QualityGate)
			}
		}
	}
}

func TestRetryFromBlocked(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/43-test", "feature", state.StatusBlocked, 2, "planner")

	if err := e.Retry("feature/43-test"); err != nil {
		t.Fatalf("Retry failed: %v", err)
	}

	ws, err := state.Load(dir, "feature/43-test")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Status != state.StatusActive {
		t.Errorf("expected status active, got %s", ws.Status)
	}
	if len(ws.Blockers) != 0 {
		t.Errorf("expected no blockers, got %d", len(ws.Blockers))
	}
}

func TestRetryFromActiveErrors(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/44-test", "feature", state.StatusActive, 2, "planner")

	err := e.Retry("feature/44-test")
	if err == nil {
		t.Fatal("expected error retrying active workflow")
	}
	if got := err.Error(); !contains(got, "not failed or blocked") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRetryFromCompletedErrors(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/45-test", "feature", state.StatusCompleted, 9, "documenter")

	err := e.Retry("feature/45-test")
	if err == nil {
		t.Fatal("expected error retrying completed workflow")
	}
	if got := err.Error(); !contains(got, "not failed or blocked") {
		t.Errorf("unexpected error: %s", got)
	}
}

// --- RollbackStep tests ---

func TestRollbackStepFromStep2(t *testing.T) {
	e, dir := setupTestEngine(t)
	createMultiStepWorkflow(t, dir, "feature/50-test", "feature", state.StatusFailed, 2)

	if err := e.RollbackStep("feature/50-test"); err != nil {
		t.Fatalf("RollbackStep failed: %v", err)
	}

	ws, err := state.Load(dir, "feature/50-test")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Status != state.StatusActive {
		t.Errorf("expected status active, got %s", ws.Status)
	}
	if ws.CurrentStep != 1 {
		t.Errorf("expected step 1, got %d", ws.CurrentStep)
	}
	if ws.CurrentRole != "human" {
		t.Errorf("expected role human (feature step 1), got %s", ws.CurrentRole)
	}
	// Step 2 records should be removed.
	for _, sr := range ws.Steps {
		if sr.Step == 2 {
			t.Error("step 2 record should have been removed")
		}
	}
	// Step 1 should have cleared completed status.
	for _, sr := range ws.Steps {
		if sr.Step == 1 && sr.Completed != "" {
			t.Errorf("step 1 completed should be cleared, got %q", sr.Completed)
		}
	}
}

func TestRollbackStepFromStep1Errors(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/51-test", "feature", state.StatusFailed, 1, "human")

	err := e.RollbackStep("feature/51-test")
	if err == nil {
		t.Fatal("expected error rolling back from step 1")
	}
	if got := err.Error(); !contains(got, "already at step 1") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRollbackStepFromCompletedErrors(t *testing.T) {
	e, dir := setupTestEngine(t)
	createWorkflowInState(t, dir, "feature/52-test", "feature", state.StatusCompleted, 9, "documenter")

	err := e.RollbackStep("feature/52-test")
	if err == nil {
		t.Fatal("expected error rolling back completed workflow")
	}
	if got := err.Error(); !contains(got, "cannot rollback") {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRollbackStepFromActiveAtStep3(t *testing.T) {
	e, dir := setupTestEngine(t)
	createMultiStepWorkflow(t, dir, "feature/53-test", "feature", state.StatusActive, 3)

	if err := e.RollbackStep("feature/53-test"); err != nil {
		t.Fatalf("RollbackStep failed: %v", err)
	}

	ws, err := state.Load(dir, "feature/53-test")
	if err != nil {
		t.Fatal(err)
	}
	if ws.CurrentStep != 2 {
		t.Errorf("expected step 2, got %d", ws.CurrentStep)
	}
	if ws.CurrentRole != "planner" {
		t.Errorf("expected role planner (feature step 2), got %s", ws.CurrentRole)
	}
}

func TestRollbackStepClearsBlockers(t *testing.T) {
	e, dir := setupTestEngine(t)
	createMultiStepWorkflow(t, dir, "feature/54-test", "feature", state.StatusBlocked, 4)
	// Manually add a blocker.
	ws, _ := state.Load(dir, "feature/54-test")
	ws.Status = state.StatusBlocked
	ws.Blockers = []state.Blocker{{Reason: "blocked", RaisedBy: "human", RaisedAt: time.Now().UTC().Format(time.RFC3339)}}
	if err := ws.Save(dir); err != nil {
		t.Fatal(err)
	}

	if err := e.RollbackStep("feature/54-test"); err != nil {
		t.Fatalf("RollbackStep failed: %v", err)
	}

	ws, err := state.Load(dir, "feature/54-test")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Status != state.StatusActive {
		t.Errorf("expected status active, got %s", ws.Status)
	}
	if len(ws.Blockers) != 0 {
		t.Errorf("expected no blockers, got %d", len(ws.Blockers))
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
