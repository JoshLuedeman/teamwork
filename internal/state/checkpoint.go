package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CheckpointState records a mid-step save-point so an agent can resume work
// on a workflow step after an interruption.
type CheckpointState struct {
	WorkflowID     string   `yaml:"workflow_id"`
	Step           int      `yaml:"step"`
	Role           string   `yaml:"role"`
	SavedAt        string   `yaml:"saved_at"`
	PartialHandoff string   `yaml:"partial_handoff,omitempty"` // path to in-progress handoff
	FilesModified  []string `yaml:"files_modified,omitempty"`
	Notes          string   `yaml:"notes,omitempty"`
}

// checkpointPath returns the filesystem path for a workflow checkpoint file.
// Slashes and backslashes in the workflow ID are replaced with hyphens to
// produce a flat filename inside .teamwork/state/.
func checkpointPath(dir, workflowID string) string {
	sanitized := strings.NewReplacer("/", "-", "\\", "-").Replace(workflowID)
	return filepath.Join(dir, ".teamwork", "state", ".checkpoint-"+sanitized+".yaml")
}

// SaveCheckpoint writes a checkpoint for the given workflow to
// .teamwork/state/.checkpoint-<sanitized-id>.yaml, creating parent
// directories as needed.
func SaveCheckpoint(dir string, cp CheckpointState) error {
	cp.SavedAt = time.Now().UTC().Format(time.RFC3339)
	p := checkpointPath(dir, cp.WorkflowID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("state: checkpoint mkdir: %w", err)
	}
	data, err := yaml.Marshal(cp)
	if err != nil {
		return fmt.Errorf("state: checkpoint marshal: %w", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("state: checkpoint write: %w", err)
	}
	return nil
}

// LoadCheckpoint reads the checkpoint for the given workflow. It returns nil
// (with no error) when no checkpoint file exists.
func LoadCheckpoint(dir, workflowID string) (*CheckpointState, error) {
	p := checkpointPath(dir, workflowID)
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: checkpoint read: %w", err)
	}
	var cp CheckpointState
	if err := yaml.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("state: checkpoint parse: %w", err)
	}
	return &cp, nil
}

// ClearCheckpoint deletes the checkpoint file for the given workflow.
// It is a no-op if no checkpoint exists.
func ClearCheckpoint(dir, workflowID string) error {
	p := checkpointPath(dir, workflowID)
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
