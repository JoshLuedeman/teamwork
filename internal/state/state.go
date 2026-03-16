// Package state manages workflow state files stored in .teamwork/state/.
// Each active workflow instance is tracked as a YAML file whose path mirrors
// the workflow ID (which may contain slashes).
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Status constants for workflow state.
const (
	StatusActive    = "active"
	StatusBlocked   = "blocked"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// WorkflowState represents the YAML state file for a single workflow instance.
type WorkflowState struct {
	ID          string       `yaml:"id" json:"id"`
	Type        string       `yaml:"type" json:"type"`
	Status      string       `yaml:"status" json:"status"`
	Goal        string       `yaml:"goal" json:"goal"`
	Issue       int          `yaml:"issue,omitempty" json:"issue,omitempty"`
	Branch      string       `yaml:"branch,omitempty" json:"branch,omitempty"`
	PullRequest int          `yaml:"pull_request,omitempty" json:"pull_request,omitempty"`
	CurrentStep int          `yaml:"current_step" json:"current_step"`
	CurrentRole string       `yaml:"current_role" json:"current_role"`
	Steps       []StepRecord `yaml:"steps" json:"steps"`
	Blockers    []Blocker    `yaml:"blockers,omitempty" json:"blockers,omitempty"`
	CreatedAt   string       `yaml:"created_at" json:"created_at"`
	UpdatedAt   string       `yaml:"updated_at" json:"updated_at"`
	CreatedBy   string       `yaml:"created_by" json:"created_by"`
}

// StepRecord captures the execution of a single workflow step.
type StepRecord struct {
	Step        int    `yaml:"step" json:"step"`
	Role        string `yaml:"role" json:"role"`
	Action      string `yaml:"action" json:"action"`
	Started     string `yaml:"started" json:"started"`
	Completed   string `yaml:"completed,omitempty" json:"completed,omitempty"`
	Handoff     string `yaml:"handoff,omitempty" json:"handoff,omitempty"`
	QualityGate string `yaml:"quality_gate,omitempty" json:"quality_gate,omitempty"`
	Repo        string `yaml:"repo,omitempty" json:"repo,omitempty"`
}

// Blocker records a reason a workflow cannot proceed.
type Blocker struct {
	Reason      string `yaml:"reason" json:"reason"`
	RaisedBy    string `yaml:"raised_by" json:"raised_by"`
	RaisedAt    string `yaml:"raised_at" json:"raised_at"`
	EscalatedTo string `yaml:"escalated_to,omitempty" json:"escalated_to,omitempty"`
}

// Filter returns the subset of workflows matching the given criteria.
// Empty values for status or workflowType are treated as "match all".
func Filter(workflows []*WorkflowState, status, workflowType string) []*WorkflowState {
	if status == "" && workflowType == "" {
		return workflows
	}

	var result []*WorkflowState
	for _, w := range workflows {
		if status != "" && w.Status != status {
			continue
		}
		if workflowType != "" && w.Type != workflowType {
			continue
		}
		result = append(result, w)
	}
	return result
}

// now returns the current UTC time formatted as RFC 3339.
func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// validateWorkflowID checks that a workflow ID does not contain path traversal.
func validateWorkflowID(id string) error {
	cleaned := filepath.Clean(id)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return fmt.Errorf("state: invalid workflow ID %q: contains path traversal", id)
	}
	return nil
}

// statePath returns the filesystem path for a workflow state file.
// Workflow IDs may contain slashes, which become subdirectories.
func statePath(dir, workflowID string) string {
	return filepath.Join(dir, ".teamwork", "state", workflowID+".yaml")
}

// New creates a new WorkflowState with status active and timestamps set to now.
// The workflowType is extracted from the ID prefix (before the first slash) if
// not provided explicitly.
func New(id, workflowType, goal string) *WorkflowState {
	ts := now()
	return &WorkflowState{
		ID:        id,
		Type:      workflowType,
		Status:    StatusActive,
		Goal:      goal,
		Steps:     []StepRecord{},
		Blockers:  []Blocker{},
		CreatedAt: ts,
		UpdatedAt: ts,
		CreatedBy: "orchestrator",
	}
}

// Load reads a workflow state file from .teamwork/state/<workflowID>.yaml.
func Load(dir, workflowID string) (*WorkflowState, error) {
	if err := validateWorkflowID(workflowID); err != nil {
		return nil, err
	}
	p := statePath(dir, workflowID)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("state: read %s: %w", p, err)
	}

	var ws WorkflowState
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("state: parse %s: %w", p, err)
	}
	return &ws, nil
}

// LoadAll reads every .yaml file under .teamwork/state/ and returns all
// parsed workflow states.
func LoadAll(dir string) ([]*WorkflowState, error) {
	root := filepath.Join(dir, ".teamwork", "state")
	var states []*WorkflowState

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("state: read %s: %w", path, err)
		}

		var ws WorkflowState
		if err := yaml.Unmarshal(data, &ws); err != nil {
			return fmt.Errorf("state: parse %s: %w", path, err)
		}
		states = append(states, &ws)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return states, nil
}

// Save writes the workflow state to .teamwork/state/<id>.yaml, creating
// parent directories as needed (workflow IDs with slashes produce subdirs).
func (s *WorkflowState) Save(dir string) error {
	if err := validateWorkflowID(s.ID); err != nil {
		return err
	}
	p := statePath(dir, s.ID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("state: mkdir %s: %w", filepath.Dir(p), err)
	}

	s.UpdatedAt = now()

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("state: write %s: %w", p, err)
	}
	return nil
}

// AdvanceStep records the completion of the current step and advances the
// workflow to the next step number. It returns an error if the workflow is
// not active or if the step number does not match current_step.
func (s *WorkflowState) AdvanceStep(step int, role, action string) error {
	if s.Status != StatusActive {
		return fmt.Errorf("state: cannot advance step: workflow %q is %s, not active", s.ID, s.Status)
	}
	if step != s.CurrentStep {
		return fmt.Errorf("state: step mismatch: expected %d, got %d", s.CurrentStep, step)
	}

	ts := now()

	// Mark the current step as completed if it exists in the log.
	for i := range s.Steps {
		if s.Steps[i].Step == step && s.Steps[i].Completed == "" {
			s.Steps[i].Completed = ts
			break
		}
	}

	// Record the new step.
	s.CurrentStep = step + 1
	s.CurrentRole = role
	s.Steps = append(s.Steps, StepRecord{
		Step:    step + 1,
		Role:    role,
		Action:  action,
		Started: ts,
	})
	s.UpdatedAt = ts
	return nil
}

// Block sets the workflow status to blocked and appends a blocker entry.
func (s *WorkflowState) Block(reason, raisedBy string) {
	s.Status = StatusBlocked
	s.Blockers = append(s.Blockers, Blocker{
		Reason:   reason,
		RaisedBy: raisedBy,
		RaisedAt: now(),
	})
	s.UpdatedAt = now()
}

// Unblock sets the workflow status back to active and clears all blockers.
func (s *WorkflowState) Unblock() {
	s.Status = StatusActive
	s.Blockers = nil
	s.UpdatedAt = now()
}

// Complete marks the workflow as completed. It returns an error if the
// workflow is not currently active.
func (s *WorkflowState) Complete() error {
	if s.Status != StatusActive {
		return fmt.Errorf("state: cannot complete: workflow %q is %s, not active", s.ID, s.Status)
	}

	ts := now()

	// Mark the last step as completed if not already done.
	for i := range s.Steps {
		if s.Steps[i].Step == s.CurrentStep && s.Steps[i].Completed == "" {
			s.Steps[i].Completed = ts
			break
		}
	}

	s.Status = StatusCompleted
	s.UpdatedAt = ts
	return nil
}

// Fail marks the workflow as failed and records the reason in the last step.
func (s *WorkflowState) Fail(reason string) {
	ts := now()

	// Record failure on the current step if present.
	for i := range s.Steps {
		if s.Steps[i].Step == s.CurrentStep && s.Steps[i].Completed == "" {
			s.Steps[i].Completed = ts
			s.Steps[i].QualityGate = "failed"
			break
		}
	}

	s.Status = StatusFailed
	s.Blockers = append(s.Blockers, Blocker{
		Reason:   reason,
		RaisedBy: s.CurrentRole,
		RaisedAt: ts,
	})
	s.UpdatedAt = ts
}

// CurrentStepStartedAt returns the start time of the current step by looking
// up its StepRecord. Returns an error if no StepRecord exists for the current
// step or if the timestamp cannot be parsed.
func (s *WorkflowState) CurrentStepStartedAt() (time.Time, error) {
	for _, sr := range s.Steps {
		if sr.Step == s.CurrentStep && sr.Completed == "" {
			t, err := time.Parse(time.RFC3339, sr.Started)
			if err != nil {
				return time.Time{}, fmt.Errorf("state: parse step start time: %w", err)
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("state: no start record for step %d", s.CurrentStep)
}

// Retry resets the current failed or blocked step so it can be re-executed.
// It clears blockers, resets the step's completion timestamp and quality gate,
// and sets the workflow status back to active.
func (s *WorkflowState) Retry() error {
	if s.Status != StatusFailed && s.Status != StatusBlocked {
		return fmt.Errorf("state: cannot retry: workflow %q is %s, must be failed or blocked", s.ID, s.Status)
	}

	ts := now()

	// Reset the current step record so it can be re-executed.
	for i := range s.Steps {
		if s.Steps[i].Step == s.CurrentStep {
			s.Steps[i].Completed = ""
			s.Steps[i].QualityGate = ""
			s.Steps[i].Started = ts
			break
		}
	}

	s.Status = StatusActive
	s.Blockers = nil
	s.UpdatedAt = ts
	return nil
}

// RollbackStep reverts the workflow to the previous step. It removes the
// current step record from the log, decrements CurrentStep, and re-opens
// the previous step by clearing its completion timestamp.
func (s *WorkflowState) RollbackStep() error {
	if s.Status != StatusActive && s.Status != StatusFailed && s.Status != StatusBlocked {
		return fmt.Errorf("state: cannot rollback: workflow %q is %s", s.ID, s.Status)
	}
	if s.CurrentStep <= 1 {
		return fmt.Errorf("state: cannot rollback: workflow %q is on step 1", s.ID)
	}

	ts := now()

	// Remove the current step record(s) from the log.
	var kept []StepRecord
	for _, sr := range s.Steps {
		if sr.Step != s.CurrentStep {
			kept = append(kept, sr)
		}
	}
	s.Steps = kept

	// Re-open the previous step by clearing its completion timestamp.
	prevStep := s.CurrentStep - 1
	for i := range s.Steps {
		if s.Steps[i].Step == prevStep {
			s.Steps[i].Completed = ""
			s.Steps[i].QualityGate = ""
		}
	}

	// Find the previous step's role from the log.
	prevRole := ""
	for i := range s.Steps {
		if s.Steps[i].Step == prevStep {
			prevRole = s.Steps[i].Role
			break
		}
	}

	s.CurrentStep = prevStep
	s.CurrentRole = prevRole
	s.Status = StatusActive
	s.Blockers = nil
	s.UpdatedAt = ts
	return nil
}

// Cancel marks the workflow as cancelled.
func (s *WorkflowState) Cancel() {
	s.Status = StatusCancelled
	s.UpdatedAt = now()
}
