package workflow

import (
	"fmt"

	"github.com/joshluedeman/teamwork/internal/config"
)

// roleTier maps agent roles to their model tier (premium, standard, fast).
// Human steps have no tier entry.
var roleTier = map[string]string{
	"planner":          "premium",
	"architect":        "premium",
	"coder":            "premium",
	"security-auditor": "premium",
	"tester":           "standard",
	"reviewer":         "standard",
	"documenter":       "fast",
	"orchestrator":     "fast",
}

// RoleTier returns the model tier for the given role, or an empty string
// if the role has no tier (e.g., "human").
func RoleTier(role string) string {
	return roleTier[role]
}

// PreviewSteps returns the step definitions for a workflow type without
// creating any state files. It is used by --dry-run to show what a workflow
// would look like before starting it.
func PreviewSteps(workflowType string) ([]StepInfo, error) {
	def, ok := definitions[workflowType]
	if !ok {
		return nil, fmt.Errorf("workflow: unknown type %q", workflowType)
	}
	// Return a copy so callers cannot mutate the shared definitions.
	steps := make([]StepInfo, len(def.Steps))
	copy(steps, def.Steps)
	return steps, nil
}

// PreviewStepsWithConfig returns step definitions for a workflow type,
// checking both built-in and custom workflow definitions from config.
func PreviewStepsWithConfig(cfg *config.Config, workflowType string) ([]StepInfo, error) {
	def, ok := definitions[workflowType]
	if !ok {
		def, ok = CustomDefinition(cfg, workflowType)
	}
	if !ok {
		return nil, fmt.Errorf("workflow: unknown type %q", workflowType)
	}
	steps := make([]StepInfo, len(def.Steps))
	copy(steps, def.Steps)
	return steps, nil
}
