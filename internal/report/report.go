// Package report builds consolidated workflow reports from state, handoff,
// and metrics data.
package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/joshluedeman/teamwork/internal/handoff"
	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/joshluedeman/teamwork/internal/state"
)

// Report is a consolidated view of a completed (or in-progress) workflow.
type Report struct {
	WorkflowID   string
	Goal         string
	Status       string
	CreatedAt    string
	CompletedAt  string
	Steps        []StepReport
	GatePassRate float64
	TotalCost    string
}

// StepReport summarises a single workflow step.
type StepReport struct {
	Step     int
	Role     string
	Status   string  // "completed" | "active" | "failed" | ...
	Duration string  // human-readable, e.g. "4m 32s"
	Handoff  string  // first paragraph of handoff artifact, or ""
	Gates    string  // "passed" | "failed" | ""
}

// Build assembles a Report for the given workflowID from the .teamwork/
// directory tree rooted at dir.
func Build(dir, workflowID string) (*Report, error) {
	ws, err := state.Load(dir, workflowID)
	if err != nil {
		return nil, fmt.Errorf("report: loading state: %w", err)
	}

	// Determine CompletedAt from the last completed step.
	completedAt := ""
	for i := len(ws.Steps) - 1; i >= 0; i-- {
		if ws.Steps[i].Completed != "" {
			completedAt = ws.Steps[i].Completed
			break
		}
	}

	// Load handoff artifacts for this workflow (best-effort).
	handoffs, _ := handoff.LoadAll(dir, workflowID)
	handoffByStep := make(map[int]*handoff.Artifact, len(handoffs))
	for _, h := range handoffs {
		handoffByStep[h.Step] = h
	}

	// Load metrics for gate pass rate and cost.
	var gateTotal, gatePassed int
	totalCost := ""
	if summaries, err := metrics.SummarizeAll(dir); err == nil {
		for _, s := range summaries {
			if s.WorkflowID == workflowID {
				totalCost = s.TotalCost
				break
			}
		}
	}
	// Gate stats come from the step records.
	for _, sr := range ws.Steps {
		if sr.QualityGate != "" {
			gateTotal++
			if sr.QualityGate == "passed" {
				gatePassed++
			}
		}
	}

	var gatePassRate float64
	if gateTotal > 0 {
		gatePassRate = float64(gatePassed) / float64(gateTotal)
	}

	// Build per-step reports.
	steps := make([]StepReport, 0, len(ws.Steps))
	for _, sr := range ws.Steps {
		duration := ""
		if sr.Started != "" && sr.Completed != "" {
			duration = formatDuration(sr.Started, sr.Completed)
		}

		firstPara := ""
		if h, ok := handoffByStep[sr.Step]; ok {
			firstPara = firstParagraph(h.Summary)
		}

		gates := sr.QualityGate // "passed" | "failed" | ""

		stepStatus := "completed"
		if sr.Completed == "" {
			stepStatus = "active"
		}

		steps = append(steps, StepReport{
			Step:     sr.Step,
			Role:     sr.Role,
			Status:   stepStatus,
			Duration: duration,
			Handoff:  firstPara,
			Gates:    gates,
		})
	}

	return &Report{
		WorkflowID:   ws.ID,
		Goal:         ws.Goal,
		Status:       ws.Status,
		CreatedAt:    ws.CreatedAt,
		CompletedAt:  completedAt,
		Steps:        steps,
		GatePassRate: gatePassRate,
		TotalCost:    totalCost,
	}, nil
}

// formatDuration computes the human-readable duration between two RFC 3339
// timestamps. Returns "" on parse error.
func formatDuration(started, completed string) string {
	t1, err := time.Parse(time.RFC3339, started)
	if err != nil {
		return ""
	}
	t2, err := time.Parse(time.RFC3339, completed)
	if err != nil {
		return ""
	}
	d := t2.Sub(t1)
	if d < 0 {
		d = -d
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m == 0 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// firstParagraph returns the first non-empty paragraph of text (up to the
// first blank line), trimmed of surrounding whitespace.
func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	var para []string
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			break
		}
		para = append(para, l)
	}
	return strings.Join(para, " ")
}
