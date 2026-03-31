// Package agentcontext assembles a rich context package for an agent step,
// pulling together role definition, previous handoff, relevant memory, ADRs,
// and open feedback into a single Markdown document.
package agentcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshluedeman/teamwork/internal/config"
	"github.com/joshluedeman/teamwork/internal/memory"
	"github.com/joshluedeman/teamwork/internal/search"
	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/joshluedeman/teamwork/internal/workflow"
)

// Package holds the assembled context for a single agent step.
type Package struct {
	WorkflowID    string
	Step          int
	Role          string
	RoleFile      string   // content of .github/agents/<role>.agent.md
	PrevHandoff   string   // content of previous step's handoff artifact
	MemoryItems   []string // relevant memory entry content strings
	RelevantADRs  []string // relevant ADR snippets
	StatusSummary string   // one-line workflow status
	OpenFeedback  []string // open feedback entries for this domain
	TokenEstimate int      // rough estimate: len(full rendered output) / 4
}

// Assemble builds a context package for the given workflow and step.
// If step is 0, the workflow's current step is used.
func Assemble(dir, workflowID string, step int) (*Package, error) {
	ws, err := state.Load(dir, workflowID)
	if err != nil {
		return nil, fmt.Errorf("agentcontext: load state: %w", err)
	}

	if step == 0 {
		step = ws.CurrentStep
	}

	// Determine the role for this step.
	role := ws.CurrentRole
	cfg, cfgErr := config.Load(dir)
	if cfgErr == nil {
		if def, ok := workflow.DefinitionFor(cfg, ws.Type); ok {
			for _, s := range def.Steps {
				if s.Number == step {
					role = s.Role
					break
				}
			}
		}
	}

	pkg := &Package{
		WorkflowID: workflowID,
		Step:       step,
		Role:       role,
	}

	// One-line status summary.
	pkg.StatusSummary = fmt.Sprintf("%s | type: %s | step: %d/%s | status: %s",
		ws.ID, ws.Type, step, role, ws.Status)

	// Read role file.
	roleFile := filepath.Join(dir, ".github", "agents", role+".agent.md")
	if data, readErr := os.ReadFile(roleFile); readErr == nil {
		pkg.RoleFile = string(data)
	}

	// Read previous step's handoff.
	if step > 1 {
		prevStep := step - 1
		for _, sr := range ws.Steps {
			if sr.Step == prevStep && sr.Handoff != "" {
				handoffPath := filepath.Join(dir, ".teamwork", "handoffs", workflowID, sr.Handoff)
				if data, readErr := os.ReadFile(handoffPath); readErr == nil {
					pkg.PrevHandoff = string(data)
				}
				break
			}
		}
	}

	// Use search to find top-5 memory/handoff results matching the goal.
	if memResults, searchErr := search.Query(dir, ws.Goal, search.QueryOptions{Type: "memory"}); searchErr == nil {
		limit := 5
		if len(memResults) < limit {
			limit = len(memResults)
		}
		for _, r := range memResults[:limit] {
			pkg.MemoryItems = append(pkg.MemoryItems, r.Snippet)
		}
	}

	// Find top-3 ADR results.
	if adrResults, searchErr := search.Query(dir, ws.Goal, search.QueryOptions{Type: "adr"}); searchErr == nil {
		limit := 3
		if len(adrResults) < limit {
			limit = len(adrResults)
		}
		for _, r := range adrResults[:limit] {
			pkg.RelevantADRs = append(pkg.RelevantADRs, r.Snippet)
		}
	}

	// Load open feedback entries for this workflow type domain.
	if ff, feedErr := memory.LoadFeedback(dir); feedErr == nil {
		for _, e := range ff.Entries {
			if e.Status != "open" {
				continue
			}
			for _, d := range e.Domain {
				if d == ws.Type {
					pkg.OpenFeedback = append(pkg.OpenFeedback, fmt.Sprintf("[%s] %s: %s", e.Date, e.Source, e.Feedback))
					break
				}
			}
		}
	}

	// Render the full Markdown document so we can estimate token count.
	rendered := pkg.Render()
	pkg.TokenEstimate = len(rendered) / 4

	return pkg, nil
}

// Render produces the assembled Markdown context document.
func (p *Package) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Agent Context: %s — Step %d (%s)\n\n", p.WorkflowID, p.Step, p.Role)
	fmt.Fprintf(&b, "**Status:** %s\n\n", p.StatusSummary)

	b.WriteString("## Role Definition\n\n")
	if p.RoleFile != "" {
		b.WriteString(p.RoleFile)
	} else {
		b.WriteString("_(role file not found)_\n")
	}
	b.WriteString("\n\n")

	b.WriteString("## Previous Handoff\n\n")
	if p.PrevHandoff != "" {
		b.WriteString(p.PrevHandoff)
	} else {
		b.WriteString("_(no previous handoff)_\n")
	}
	b.WriteString("\n\n")

	b.WriteString("## Relevant Memory\n\n")
	if len(p.MemoryItems) == 0 {
		b.WriteString("_(no relevant memory entries)_\n")
	} else {
		for i, item := range p.MemoryItems {
			fmt.Fprintf(&b, "### Memory %d\n\n%s\n\n", i+1, item)
		}
	}

	b.WriteString("## Relevant ADRs\n\n")
	if len(p.RelevantADRs) == 0 {
		b.WriteString("_(no relevant ADRs)_\n")
	} else {
		for i, adr := range p.RelevantADRs {
			fmt.Fprintf(&b, "### ADR %d\n\n%s\n\n", i+1, adr)
		}
	}

	b.WriteString("## Open Feedback\n\n")
	if len(p.OpenFeedback) == 0 {
		b.WriteString("_(no open feedback)_\n")
	} else {
		for _, f := range p.OpenFeedback {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}

	return b.String()
}
