package state

import (
"testing"
"time"
)

func TestCurrentStepStartedAt(t *testing.T) {
started := time.Now().UTC().Add(-5 * time.Second).Format(time.RFC3339)
ws := &WorkflowState{
ID:          "test/1-example",
CurrentStep: 2,
Steps: []StepRecord{
{Step: 1, Role: "planner", Started: "2025-01-01T00:00:00Z", Completed: "2025-01-01T00:01:00Z"},
{Step: 2, Role: "coder", Started: started},
},
}

got, err := ws.CurrentStepStartedAt()
if err != nil {
t.Fatalf("CurrentStepStartedAt() error: %v", err)
}

expected, _ := time.Parse(time.RFC3339, started)
if !got.Equal(expected) {
t.Errorf("CurrentStepStartedAt() = %v, want %v", got, expected)
}
}

func TestCurrentStepStartedAt_NoRecord(t *testing.T) {
ws := &WorkflowState{
ID:          "test/1-example",
CurrentStep: 3,
Steps: []StepRecord{
{Step: 1, Role: "planner", Started: "2025-01-01T00:00:00Z", Completed: "2025-01-01T00:01:00Z"},
},
}

_, err := ws.CurrentStepStartedAt()
if err == nil {
t.Fatal("expected error for missing step record, got nil")
}
}

func TestCurrentStepStartedAt_SkipsCompleted(t *testing.T) {
// If the current step has a completed record and an incomplete one,
// only the incomplete one should be returned.
ws := &WorkflowState{
ID:          "test/1-example",
CurrentStep: 2,
Steps: []StepRecord{
{Step: 2, Role: "coder", Started: "2025-01-01T00:00:00Z", Completed: "2025-01-01T00:01:00Z"},
{Step: 2, Role: "coder", Started: "2025-01-01T00:05:00Z"},
},
}

got, err := ws.CurrentStepStartedAt()
if err != nil {
t.Fatalf("CurrentStepStartedAt() error: %v", err)
}

expected, _ := time.Parse(time.RFC3339, "2025-01-01T00:05:00Z")
if !got.Equal(expected) {
t.Errorf("CurrentStepStartedAt() = %v, want %v", got, expected)
}
}

func TestStartCreatesStepRecord(t *testing.T) {
// Verify that New creates an empty steps slice and that callers
// (like Engine.Start) should add a record for step 1.
ws := New("test/1-example", "feature", "Test goal")
if len(ws.Steps) != 0 {
t.Errorf("New() should create empty steps, got %d", len(ws.Steps))
}

// Simulate what Engine.Start now does: add a StepRecord for step 1.
ws.CurrentStep = 1
ws.CurrentRole = "planner"
ws.Steps = append(ws.Steps, StepRecord{
Step:    1,
Role:    "planner",
Action:  "Create feature request",
Started: ws.CreatedAt,
})

got, err := ws.CurrentStepStartedAt()
if err != nil {
t.Fatalf("CurrentStepStartedAt() after adding step record: %v", err)
}

expected, _ := time.Parse(time.RFC3339, ws.CreatedAt)
if !got.Equal(expected) {
t.Errorf("start time = %v, want %v", got, expected)
}
}

func TestAdvanceStepSetsStartTime(t *testing.T) {
ws := New("test/1-example", "feature", "Test goal")
ws.CurrentStep = 1
ws.CurrentRole = "planner"
ws.Steps = append(ws.Steps, StepRecord{
Step:    1,
Role:    "planner",
Action:  "Plan",
Started: ws.CreatedAt,
})

before := time.Now().UTC().Truncate(time.Second)
if err := ws.AdvanceStep(1, "coder", "Implement"); err != nil {
t.Fatalf("AdvanceStep: %v", err)
}
after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

got, err := ws.CurrentStepStartedAt()
if err != nil {
t.Fatalf("CurrentStepStartedAt() after advance: %v", err)
}

if got.Before(before) || got.After(after) {
t.Errorf("step 2 start time %v not between %v and %v", got, before, after)
}
}

func TestFilter(t *testing.T) {
workflows := []*WorkflowState{
{ID: "feat-1", Type: "feature", Status: StatusActive},
{ID: "feat-2", Type: "feature", Status: StatusCompleted},
{ID: "bug-1", Type: "bugfix", Status: StatusActive},
{ID: "bug-2", Type: "bugfix", Status: StatusBlocked},
{ID: "hot-1", Type: "hotfix", Status: StatusFailed},
{ID: "ref-1", Type: "refactor", Status: StatusCancelled},
}

tests := []struct {
name         string
status       string
workflowType string
wantIDs      []string
}{
{
name:    "no filters returns all",
wantIDs: []string{"feat-1", "feat-2", "bug-1", "bug-2", "hot-1", "ref-1"},
},
{
name:    "filter by active status",
status:  StatusActive,
wantIDs: []string{"feat-1", "bug-1"},
},
{
name:    "filter by blocked status",
status:  StatusBlocked,
wantIDs: []string{"bug-2"},
},
{
name:    "filter by completed status",
status:  StatusCompleted,
wantIDs: []string{"feat-2"},
},
{
name:    "filter by failed status",
status:  StatusFailed,
wantIDs: []string{"hot-1"},
},
{
name:    "filter by cancelled status",
status:  StatusCancelled,
wantIDs: []string{"ref-1"},
},
{
name:         "filter by type feature",
workflowType: "feature",
wantIDs:      []string{"feat-1", "feat-2"},
},
{
name:         "filter by type bugfix",
workflowType: "bugfix",
wantIDs:      []string{"bug-1", "bug-2"},
},
{
name:         "filter by status and type",
status:       StatusActive,
workflowType: "feature",
wantIDs:      []string{"feat-1"},
},
{
name:         "filter returns empty when no match",
status:       StatusActive,
workflowType: "hotfix",
wantIDs:      nil,
},
{
name:         "filter by nonexistent type returns empty",
workflowType: "nonexistent",
wantIDs:      nil,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
got := Filter(workflows, tt.status, tt.workflowType)

if len(got) != len(tt.wantIDs) {
t.Fatalf("Filter() returned %d workflows, want %d", len(got), len(tt.wantIDs))
}

for i, w := range got {
if w.ID != tt.wantIDs[i] {
t.Errorf("Filter()[%d].ID = %q, want %q", i, w.ID, tt.wantIDs[i])
}
}
})
}
}

func TestFilter_NilSlice(t *testing.T) {
got := Filter(nil, "active", "feature")
if got != nil {
t.Errorf("Filter(nil, ...) = %v, want nil", got)
}
}

func TestFilter_EmptySlice(t *testing.T) {
got := Filter([]*WorkflowState{}, "", "")
if len(got) != 0 {
t.Errorf("Filter([], ...) = %d items, want 0", len(got))
}
}
