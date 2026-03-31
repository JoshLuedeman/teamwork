// Package workflow provides the workflow state machine engine.
//
// It ties together state, config, handoff, and metrics to manage workflow
// execution through its lifecycle: start, advance, block/unblock, and complete.
package workflow

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/joshluedeman/teamwork/internal/gates"
	"github.com/joshluedeman/teamwork/internal/handoff"
	"github.com/joshluedeman/teamwork/internal/memory"
	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/joshluedeman/teamwork/internal/state"
)

// Engine manages workflow execution by coordinating state, handoffs, and metrics.
type Engine struct {
	Dir        string         // project root directory
	Config     *config.Config // parsed .teamwork/config.yaml
	GateRunner gates.Runner   // optional runner for extra quality gates
}

// StepInfo describes a single step within a workflow definition.
type StepInfo struct {
	Number int
	Role   string
	Action string
}

// WorkflowDefinition describes the step sequence for a workflow type.
type WorkflowDefinition struct {
	Type  string
	Steps []StepInfo
}

// NextAction describes what should happen next for an active workflow.
type NextAction struct {
	WorkflowID string
	Step       int
	Role       string
	Action     string
	Context    string // summary from previous handoff
	Repo       string // target repo name (empty = hub repo)
}

// Workflow definitions keyed by type, derived from state-machines.md.
var definitions = map[string]WorkflowDefinition{
	"feature": {
		Type: "feature",
		Steps: []StepInfo{
			{1, "human", "Create feature request"},
			{2, "planner", "Decompose goal into tasks"},
			{3, "architect", "Review feasibility, design"},
			{4, "coder", "Implement and open PR"},
			{5, "tester", "Validate acceptance criteria"},
			{6, "security-auditor", "Scan for vulnerabilities"},
			{7, "reviewer", "Review for quality"},
			{8, "human", "Approve and merge PR"},
			{9, "documenter", "Update docs and changelog"},
		},
	},
	"bugfix": {
		Type: "bugfix",
		Steps: []StepInfo{
			{1, "human", "File bug report"},
			{2, "planner", "Confirm reproduction, create fix task"},
			{3, "architect", "Evaluate design implications"},
			{4, "coder", "Write regression test and fix"},
			{5, "tester", "Validate fix, check regressions"},
			{6, "security-auditor", "Assess fix security"},
			{7, "reviewer", "Review fix correctness"},
			{8, "human", "Approve and merge PR"},
			{9, "documenter", "Update changelog"},
		},
	},
	"refactor": {
		Type: "refactor",
		Steps: []StepInfo{
			{1, "human", "Identify refactoring need"},
			{2, "architect", "Define scope and approach"},
			{3, "planner", "Break into incremental steps"},
			{4, "coder", "Implement and update tests"},
			{5, "tester", "Validate behavior unchanged"},
			{6, "reviewer", "Review correctness"},
			{7, "human", "Approve and merge PR"},
		},
	},
	"hotfix": {
		Type: "hotfix",
		Steps: []StepInfo{
			{1, "human", "Report production incident"},
			{2, "coder", "Implement minimal fix"},
			{3, "tester", "Validate fix"},
			{4, "security-auditor", "Check security implications"},
			{5, "reviewer", "Fast-track review"},
			{6, "human", "Approve, merge, deploy"},
			{7, "documenter", "Update changelog, postmortem stub"},
		},
	},
	"security-response": {
		Type: "security-response",
		Steps: []StepInfo{
			{1, "human", "Assess severity and scope"},
			{2, "architect", "Determine remediation approach"},
			{3, "coder", "Implement fix on private branch"},
			{4, "tester", "Validate fix"},
			{5, "security-auditor", "Verify remediation complete"},
			{6, "reviewer", "Review fix"},
			{7, "human", "Merge and decide disclosure"},
			{8, "documenter", "Publish advisory"},
		},
	},
	"spike": {
		Type: "spike",
		Steps: []StepInfo{
			{1, "human", "Identify question, set time box"},
			{2, "planner", "Scope investigation"},
			{3, "architect", "Research and document findings"},
			{4, "reviewer", "Evaluate recommendation"},
			{5, "human", "Decide approach"},
		},
	},
	"release": {
		Type: "release",
		Steps: []StepInfo{
			{1, "human", "Initiate release"},
			{2, "planner", "Compile inclusion list"},
			{3, "tester", "Run regression suite"},
			{4, "security-auditor", "Final security scan"},
			{5, "documenter", "Finalize changelog"},
			{6, "coder", "Create release branch/tag"},
			{7, "reviewer", "Verify changelog and version"},
			{8, "human", "Publish release"},
		},
	},
	"rollback": {
		Type: "rollback",
		Steps: []StepInfo{
			{1, "human", "Identify bad merge"},
			{2, "human", "Decide revert vs forward-fix"},
			{3, "coder", "Create revert PR"},
			{4, "tester", "Validate revert"},
			{5, "reviewer", "Fast-track review"},
			{6, "human", "Merge revert PR"},
			{7, "documenter", "File follow-up, update changelog"},
		},
	},
	"dependency-update": {
		Type: "dependency-update",
		Steps: []StepInfo{
			{1, "human", "Identify update need"},
			{2, "coder", "Evaluate changelog and breaking changes"},
			{3, "coder", "Update dependency and adapt code"},
			{4, "tester", "Run full test suite"},
			{5, "security-auditor", "Check for vulnerabilities"},
			{6, "reviewer", "Review version bump"},
			{7, "human", "Approve and merge PR"},
		},
	},
	"documentation": {
		Type: "documentation",
		Steps: []StepInfo{
			{1, "human", "Identify documentation gap"},
			{2, "documenter", "Assess scope, draft outline"},
			{3, "documenter", "Write or update docs, open PR"},
			{4, "reviewer", "Review for accuracy and clarity"},
			{5, "human", "Approve and merge PR"},
		},
	},
}

// NewEngine creates an Engine and loads the project config from dir.
func NewEngine(dir string) (*Engine, error) {
	cfg, err := config.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("workflow: load config: %w", err)
	}
	return &Engine{Dir: dir, Config: cfg, GateRunner: gates.ShellRunner{}}, nil
}

// lookupDefinition returns the WorkflowDefinition for the given type,
// checking built-in definitions first and then custom workflows from config.
func (e *Engine) lookupDefinition(wfType string) (WorkflowDefinition, bool) {
	if def, ok := definitions[wfType]; ok {
		return def, true
	}
	return CustomDefinition(e.Config, wfType)
}

// CustomDefinition builds a WorkflowDefinition from a custom workflow in config.
// It returns the definition and true if found, or a zero value and false otherwise.
func CustomDefinition(cfg *config.Config, wfType string) (WorkflowDefinition, bool) {
	if cfg == nil || cfg.CustomWorkflows == nil {
		return WorkflowDefinition{}, false
	}
	cw, ok := cfg.CustomWorkflows[wfType]
	if !ok || len(cw.Steps) == 0 {
		return WorkflowDefinition{}, false
	}
	def := WorkflowDefinition{Type: wfType}
	for i, s := range cw.Steps {
		def.Steps = append(def.Steps, StepInfo{
			Number: i + 1,
			Role:   s.Role,
			Action: s.Description,
		})
	}
	return def, true
}

// IsBuiltinType reports whether the given type is a built-in workflow type.
func IsBuiltinType(wfType string) bool {
	_, ok := definitions[wfType]
	return ok
}

// DefinitionFor returns the workflow definition for the given type,
// checking built-in definitions first and then custom config definitions.
// Returns false if the type is not known.
func DefinitionFor(cfg *config.Config, wfType string) (WorkflowDefinition, bool) {
	if def, ok := definitions[wfType]; ok {
		return def, true
	}
	return CustomDefinition(cfg, wfType)
}

// Start initializes a new workflow. It generates an ID from the workflow type,
// issue number, and goal, creates the state file, and logs a start metric.
func (e *Engine) Start(workflowType, goal string, issue int) (*state.WorkflowState, error) {
	def, ok := e.lookupDefinition(workflowType)
	if !ok {
		return nil, fmt.Errorf("workflow: unknown type %q", workflowType)
	}

	id := generateID(workflowType, goal, issue)
	ws := state.New(id, workflowType, goal)
	ws.Issue = issue
	ws.Branch = id
	ws.CurrentStep = 1
	ws.CurrentRole = def.Steps[0].Role

	// Record a StepRecord for step 1 so its start time is available later.
	ws.Steps = append(ws.Steps, state.StepRecord{
		Step:    1,
		Role:    ws.CurrentRole,
		Action:  def.Steps[0].Action,
		Started: ws.CreatedAt,
	})

	if err := ws.Save(e.Dir); err != nil {
		return nil, fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.LogStart(e.Dir, id, 1, ws.CurrentRole, goal); err != nil {
		return nil, fmt.Errorf("workflow: log start: %w", err)
	}

	return ws, nil
}

// Next scans all active workflows and returns the next action for each.
func (e *Engine) Next() ([]NextAction, error) {
	states, err := state.LoadAll(e.Dir)
	if err != nil {
		return nil, fmt.Errorf("workflow: load states: %w", err)
	}

	var actions []NextAction
	for _, ws := range states {
		if ws.Status != state.StatusActive {
			continue
		}

		def, ok := e.lookupDefinition(ws.Type)
		if !ok {
			continue
		}

		// Find the current step definition.
		var stepInfo *StepInfo
		for i := range def.Steps {
			if def.Steps[i].Number == ws.CurrentStep {
				stepInfo = &def.Steps[i]
				break
			}
		}
		if stepInfo == nil {
			continue
		}

		na := NextAction{
			WorkflowID: ws.ID,
			Step:       ws.CurrentStep,
			Role:       stepInfo.Role,
			Action:     stepInfo.Action,
		}

		// Attach repo and context from the current/previous step records.
		for i := len(ws.Steps) - 1; i >= 0; i-- {
			if ws.Steps[i].Step == ws.CurrentStep && ws.Steps[i].Repo != "" {
				na.Repo = ws.Steps[i].Repo
				break
			}
		}
		if ws.CurrentStep > 1 {
			prevStep := ws.CurrentStep - 1
			for i := len(ws.Steps) - 1; i >= 0; i-- {
				if ws.Steps[i].Step == prevStep && ws.Steps[i].Handoff != "" {
					na.Context = ws.Steps[i].Handoff
					break
				}
			}
		}

		// Surface any saved checkpoint for this step.
		if cp, _ := state.LoadCheckpoint(e.Dir, ws.ID); cp != nil && cp.Step == ws.CurrentStep {
			note := fmt.Sprintf("[checkpoint: step %d saved at %s", cp.Step, cp.SavedAt)
			if len(cp.FilesModified) > 0 {
				note += fmt.Sprintf(", files: %s", strings.Join(cp.FilesModified, ", "))
			}
			if cp.Notes != "" {
				note += ", notes: " + cp.Notes
			}
			note += "]"
			if na.Context != "" {
				na.Context = na.Context + "\n" + note
			} else {
				na.Context = note
			}
		}

		actions = append(actions, na)
	}

	return actions, nil
}

// Handoff validates a handoff artifact, saves it, runs any extra quality gates,
// advances the workflow state, and logs completion and start metrics for the
// transition.
func (e *Engine) Handoff(workflowID string, artifact *handoff.Artifact) error {
	if errs := handoff.Validate(artifact); errs != nil {
		return fmt.Errorf("workflow: invalid handoff: %s", strings.Join(errs, "; "))
	}

	// Enforce quality gates from config.
	if err := e.enforceQualityGates(workflowID, artifact); err != nil {
		return err
	}

	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status != state.StatusActive {
		return fmt.Errorf("workflow: cannot handoff: workflow %q is %s", workflowID, ws.Status)
	}

	if err := artifact.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save handoff: %w", err)
	}

	// Calculate the elapsed time for the current step.
	var durationSec int
	if startedAt, err := ws.CurrentStepStartedAt(); err == nil {
		durationSec = int(time.Since(startedAt).Seconds())
	}

	// Run extra quality gates if configured.
	if e.GateRunner != nil && e.Config != nil {
		location := gates.GateKey(ws.CurrentStep)
		conditions := gates.Lookup(e.Config.Workflows.ExtraGates, ws.Type, location)
		if len(conditions) > 0 {
			_, passed, err := gates.RunGate(location, conditions, e.Dir, e.GateRunner)
			if err != nil {
				return fmt.Errorf("workflow: run extra gate: %w", err)
			}
			if !passed {
				return fmt.Errorf("workflow: extra gate failed at %s", location)
			}
		}
	}

	// Run custom gates from config.QualityGates.Custom.
	if e.Config != nil && len(e.Config.QualityGates.Custom) > 0 {
		customGateResult := "passed"
		for _, cg := range e.Config.QualityGates.Custom {
			// Skip if OnStep is non-empty and current step is not listed.
			if len(cg.OnStep) > 0 {
				match := false
				for _, s := range cg.OnStep {
					if s == ws.CurrentStep {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}
			runner := e.GateRunner
			if runner == nil {
				runner = gates.ShellRunner{}
			}
			_, passed, err := gates.RunGate(cg.Name, []string{cg.Command}, e.Dir, runner)
			if err != nil {
				return fmt.Errorf("workflow: run custom gate %q: %w", cg.Name, err)
			}
			if !passed {
				customGateResult = "failed"
				// Record failure on the step before returning.
				for i := range ws.Steps {
					if ws.Steps[i].Step == ws.CurrentStep && ws.Steps[i].Completed == "" {
						ws.Steps[i].CustomGates = customGateResult
						break
					}
				}
				_ = ws.Save(e.Dir)
				return fmt.Errorf("workflow: custom gate %q failed at step %d", cg.Name, ws.CurrentStep)
			}
		}
		// Record successful custom gate result on the current step.
		for i := range ws.Steps {
			if ws.Steps[i].Step == ws.CurrentStep && ws.Steps[i].Completed == "" {
				ws.Steps[i].CustomGates = customGateResult
				break
			}
		}
	}

	// Log completion of the current step.
	if err := metrics.LogComplete(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, artifact.Summary, durationSec); err != nil {
		return fmt.Errorf("workflow: log complete: %w", err)
	}

	// Advance state to the next step.
	nextRole := artifact.NextRole
	nextAction := ""
	def, ok := e.lookupDefinition(ws.Type)
	if ok {
		for _, s := range def.Steps {
			if s.Number == ws.CurrentStep+1 {
				nextRole = s.Role
				nextAction = s.Action
				break
			}
		}
	}

	handoffFile := fmt.Sprintf("%02d-%s.md", artifact.Step, artifact.Role)
	if err := ws.AdvanceStep(ws.CurrentStep, nextRole, nextAction); err != nil {
		return fmt.Errorf("workflow: advance step: %w", err)
	}

	// Record handoff filename on the completed step.
	for i := range ws.Steps {
		if ws.Steps[i].Step == artifact.Step && ws.Steps[i].Handoff == "" {
			ws.Steps[i].Handoff = handoffFile
			break
		}
	}

	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	// If this is a reviewer handoff containing "changes requested", capture feedback.
	if strings.ToLower(artifact.Role) == "reviewer" &&
		strings.Contains(strings.ToLower(artifact.Render()), "changes requested") {
		_ = memory.AppendFeedback(e.Dir, memory.FeedbackEntry{
			Source:   workflowID,
			Domain:   []string{ws.Type},
			Feedback: artifact.Summary,
			Status:   "open",
		})
	}

	// Clear any checkpoint for this workflow now that the step advanced.
	_ = state.ClearCheckpoint(e.Dir, workflowID)

	// Log start of the next step.
	if err := metrics.LogStart(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, nextAction); err != nil {
		return fmt.Errorf("workflow: log next start: %w", err)
	}

	return nil
}

// enforceQualityGates checks each configured quality gate against the
// artifact's GateResults and logs the outcome via metrics.LogGate. It returns
// an error describing the first gate that has not passed, or nil if all
// required gates are satisfied.
func (e *Engine) enforceQualityGates(workflowID string, artifact *handoff.Artifact) error {
	qgCfg := e.Config.QualityGates

	type gate struct {
		name    string
		enabled bool
	}

	checks := []gate{
		{"tests_pass", qgCfg.TestsPass},
		{"lint_pass", qgCfg.LintPass},
	}

	for _, g := range checks {
		if !g.enabled {
			continue
		}
		passed, reported := artifact.GateResults[g.name]
		if reported && passed {
			_ = metrics.LogGate(e.Dir, workflowID, artifact.Step, artifact.Role, g.name, "passed")
			continue
		}
		_ = metrics.LogGate(e.Dir, workflowID, artifact.Step, artifact.Role, g.name, "failed")
		if !reported {
			return fmt.Errorf("workflow: quality gate %q required but not reported in handoff", g.name)
		}
		return fmt.Errorf("workflow: quality gate %q failed", g.name)
	}

	// Run secrets gate if enabled.
	if qgCfg.SecretsGate {
		runner := e.GateRunner
		if runner == nil {
			runner = gates.ShellRunner{}
		}
		found, details, err := gates.RunSecretsGate(e.Dir, runner)
		if err != nil {
			return fmt.Errorf("workflow: secrets gate: %w", err)
		}
		if found {
			_ = metrics.LogGate(e.Dir, workflowID, artifact.Step, artifact.Role, "secrets_scan", "failed: "+details)
			return fmt.Errorf("workflow: secrets gate: secrets detected in repository")
		}
		_ = metrics.LogGate(e.Dir, workflowID, artifact.Step, artifact.Role, "secrets_scan", "passed")
	}

	return nil
}

// Approve records a human approval (quality gate pass) and advances the
// workflow to the next step.
func (e *Engine) Approve(workflowID string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status != state.StatusActive {
		return fmt.Errorf("workflow: cannot approve: workflow %q is %s", workflowID, ws.Status)
	}

	if err := metrics.LogGate(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, "Human approval", "passed"); err != nil {
		return fmt.Errorf("workflow: log gate: %w", err)
	}

	// Mark quality gate on current step.
	for i := range ws.Steps {
		if ws.Steps[i].Step == ws.CurrentStep {
			ws.Steps[i].QualityGate = "passed"
			break
		}
	}

	// Advance to next step if one exists.
	def, ok := e.lookupDefinition(ws.Type)
	if ok {
		var next *StepInfo
		for i := range def.Steps {
			if def.Steps[i].Number == ws.CurrentStep+1 {
				next = &def.Steps[i]
				break
			}
		}
		if next != nil {
			// Calculate the elapsed time for the current step.
			var durationSec int
			if startedAt, err := ws.CurrentStepStartedAt(); err == nil {
				durationSec = int(time.Since(startedAt).Seconds())
			}
			// Log completion of the current step.
			if err := metrics.LogComplete(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, "Human approval", durationSec); err != nil {
				return fmt.Errorf("workflow: log complete: %w", err)
			}
			if err := ws.AdvanceStep(ws.CurrentStep, next.Role, next.Action); err != nil {
				return fmt.Errorf("workflow: advance step: %w", err)
			}
			// Log start of the next step.
			if err := metrics.LogStart(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, next.Action); err != nil {
				return fmt.Errorf("workflow: log next start: %w", err)
			}
		}
	}

	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}
	return nil
}

// Block marks a workflow as blocked with the given reason and role.
func (e *Engine) Block(workflowID, reason, role string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	ws.Block(reason, role)
	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.Log(e.Dir, workflowID, metrics.Event{
		Step:   ws.CurrentStep,
		Role:   role,
		Action: metrics.ActionBlock,
		Detail: reason,
	}); err != nil {
		return fmt.Errorf("workflow: log block: %w", err)
	}

	return nil
}

// Unblock removes all blockers and sets the workflow back to active.
func (e *Engine) Unblock(workflowID string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status != state.StatusBlocked {
		return fmt.Errorf("workflow: cannot unblock: workflow %q is %s, not blocked", workflowID, ws.Status)
	}

	ws.Unblock()
	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.Log(e.Dir, workflowID, metrics.Event{
		Step:   ws.CurrentStep,
		Role:   ws.CurrentRole,
		Action: metrics.ActionUnblock,
		Detail: "Blockers resolved",
	}); err != nil {
		return fmt.Errorf("workflow: log unblock: %w", err)
	}

	return nil
}

// Complete marks a workflow as completed after validating that all steps are done.
func (e *Engine) Complete(workflowID string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	def, ok := e.lookupDefinition(ws.Type)
	if ok {
		totalSteps := len(def.Steps)
		if ws.CurrentStep < totalSteps {
			return fmt.Errorf("workflow: cannot complete: only on step %d of %d", ws.CurrentStep, totalSteps)
		}
	}

	// Calculate the elapsed time for the final step before marking it complete,
	// since ws.Complete() sets Completed on the StepRecord and
	// CurrentStepStartedAt() only matches records where Completed == "".
	var durationSec int
	if startedAt, err := ws.CurrentStepStartedAt(); err == nil {
		durationSec = int(time.Since(startedAt).Seconds())
	}

	if err := ws.Complete(); err != nil {
		return fmt.Errorf("workflow: complete: %w", err)
	}
	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.LogComplete(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, "Workflow completed", durationSec); err != nil {
		return fmt.Errorf("workflow: log complete: %w", err)
	}

	return nil
}

// Cancel marks a workflow as cancelled with an optional reason.
func (e *Engine) Cancel(workflowID, reason string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status == state.StatusCompleted || ws.Status == state.StatusCancelled || ws.Status == state.StatusFailed {
		return fmt.Errorf("workflow: cannot cancel: workflow %q is already %s", workflowID, ws.Status)
	}

	ws.Cancel()
	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.Log(e.Dir, workflowID, metrics.Event{
		Step:   ws.CurrentStep,
		Role:   ws.CurrentRole,
		Action: metrics.ActionCancel,
		Detail: reason,
	}); err != nil {
		return fmt.Errorf("workflow: log cancel: %w", err)
	}

	return nil
}

// Fail marks a workflow as failed with a required reason.
func (e *Engine) Fail(workflowID, reason string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status == state.StatusCompleted || ws.Status == state.StatusCancelled || ws.Status == state.StatusFailed {
		return fmt.Errorf("workflow: cannot fail: workflow %q is already %s", workflowID, ws.Status)
	}

	ws.Fail(reason)
	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.LogFail(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, reason, reason); err != nil {
		return fmt.Errorf("workflow: log fail: %w", err)
	}

	return nil
}

// Retry resets the current failed or blocked step so it can be re-run.
// It clears the step's completion timestamp, quality gate, and any blockers,
// then sets the workflow back to active.
func (e *Engine) Retry(workflowID string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status != state.StatusFailed && ws.Status != state.StatusBlocked {
		return fmt.Errorf("workflow: cannot retry: workflow %q is %s, not failed or blocked", workflowID, ws.Status)
	}

	// Reset the current step record so it can be re-executed.
	for i := range ws.Steps {
		if ws.Steps[i].Step == ws.CurrentStep {
			ws.Steps[i].Completed = ""
			ws.Steps[i].QualityGate = ""
		}
	}

	ws.Status = state.StatusActive
	ws.Blockers = nil
	ws.UpdatedAt = ""

	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.LogRetry(e.Dir, workflowID, ws.CurrentStep, ws.CurrentRole, "Retrying step"); err != nil {
		return fmt.Errorf("workflow: log retry: %w", err)
	}

	return nil
}

// RollbackStep reverts the workflow to the previous step. It removes the
// current step record, decrements the step counter, and sets the workflow
// back to active. Rollback from step 1 is not allowed.
func (e *Engine) RollbackStep(workflowID string) error {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return fmt.Errorf("workflow: load state: %w", err)
	}

	if ws.Status == state.StatusCompleted || ws.Status == state.StatusCancelled {
		return fmt.Errorf("workflow: cannot rollback: workflow %q is %s", workflowID, ws.Status)
	}

	if ws.CurrentStep <= 1 {
		return fmt.Errorf("workflow: cannot rollback: workflow %q is already at step 1", workflowID)
	}

	fromStep := ws.CurrentStep

	// Remove step records for the current step.
	var kept []state.StepRecord
	for _, sr := range ws.Steps {
		if sr.Step != ws.CurrentStep {
			kept = append(kept, sr)
		}
	}
	ws.Steps = kept

	// Move to the previous step and reset its completion status.
	ws.CurrentStep = fromStep - 1
	for i := range ws.Steps {
		if ws.Steps[i].Step == ws.CurrentStep {
			ws.Steps[i].Completed = ""
			ws.Steps[i].QualityGate = ""
		}
	}

	// Resolve the role from the workflow definition.
	def, ok := definitions[ws.Type]
	if ok {
		for _, s := range def.Steps {
			if s.Number == ws.CurrentStep {
				ws.CurrentRole = s.Role
				break
			}
		}
	}

	ws.Status = state.StatusActive
	ws.Blockers = nil

	if err := ws.Save(e.Dir); err != nil {
		return fmt.Errorf("workflow: save state: %w", err)
	}

	if err := metrics.LogRollback(e.Dir, workflowID, fromStep, ws.CurrentStep, ws.CurrentRole, "Rolled back step"); err != nil {
		return fmt.Errorf("workflow: log rollback: %w", err)
	}

	return nil
}

// Status returns all workflow states across the project.
func (e *Engine) Status() ([]*state.WorkflowState, error) {
	states, err := state.LoadAll(e.Dir)
	if err != nil {
		return nil, fmt.Errorf("workflow: load states: %w", err)
	}
	return states, nil
}

// History returns the full history for a workflow: its state and all handoff artifacts.
func (e *Engine) History(workflowID string) (*state.WorkflowState, []*handoff.Artifact, error) {
	ws, err := state.Load(e.Dir, workflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow: load state: %w", err)
	}

	artifacts, err := handoff.LoadAll(e.Dir, workflowID)
	if err != nil {
		return ws, nil, fmt.Errorf("workflow: load handoffs: %w", err)
	}

	return ws, artifacts, nil
}

// nonAlphaNum matches any character that is not alphanumeric or a hyphen.
var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

// multiHyphen matches consecutive hyphens.
var multiHyphen = regexp.MustCompile(`-{2,}`)

// generateID creates a workflow ID slug like "feature/42-add-oauth" from
// the workflow type, issue number, and goal text. The goal is converted to
// kebab-case and truncated to 40 characters.
func generateID(workflowType, goal string, issue int) string {
	slug := strings.ToLower(goal)
	slug = nonAlphaNum.ReplaceAllString(slug, "-")
	slug = multiHyphen.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}

	if issue > 0 {
		return fmt.Sprintf("%s/%d-%s", workflowType, issue, slug)
	}
	return fmt.Sprintf("%s/%s", workflowType, slug)
}
