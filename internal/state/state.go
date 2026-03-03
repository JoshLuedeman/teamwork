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
	ID          string       `yaml:"id"`
	Type        string       `yaml:"type"`
	Status      string       `yaml:"status"`
	Goal        string       `yaml:"goal"`
	Issue       int          `yaml:"issue,omitempty"`
	Branch      string       `yaml:"branch,omitempty"`
	PullRequest int          `yaml:"pull_request,omitempty"`
	CurrentStep int          `yaml:"current_step"`
	CurrentRole string       `yaml:"current_role"`
	Steps       []StepRecord `yaml:"steps"`
	Blockers    []Blocker    `yaml:"blockers,omitempty"`
	CreatedAt   string       `yaml:"created_at"`
	UpdatedAt   string       `yaml:"updated_at"`
	CreatedBy   string       `yaml:"created_by"`
}

// StepRecord captures the execution of a single workflow step.
type StepRecord struct {
	Step        int    `yaml:"step"`
	Role        string `yaml:"role"`
	Action      string `yaml:"action"`
	Started     string `yaml:"started"`
	Completed   string `yaml:"completed,omitempty"`
	Handoff     string `yaml:"handoff,omitempty"`
	QualityGate string `yaml:"quality_gate,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
}

// Blocker records a reason a workflow cannot proceed.
type Blocker struct {
	Reason      string `yaml:"reason"`
	RaisedBy    string `yaml:"raised_by"`
	RaisedAt    string `yaml:"raised_at"`
	EscalatedTo string `yaml:"escalated_to,omitempty"`
}

// now returns the current UTC time formatted as RFC 3339.
func now() string {
	return time.Now().UTC().Format(time.RFC3339)
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

// Cancel marks the workflow as cancelled.
func (s *WorkflowState) Cancel() {
	s.Status = StatusCancelled
	s.UpdatedAt = now()
}
