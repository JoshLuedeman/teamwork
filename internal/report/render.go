package report

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderMarkdown returns a clean Markdown document suitable for a PR comment
// or workflow summary.
func RenderMarkdown(r *Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Workflow Report: %s\n\n", r.WorkflowID)
	fmt.Fprintf(&b, "**Goal:** %s\n\n", r.Goal)
	fmt.Fprintf(&b, "**Status:** %s\n\n", r.Status)
	if r.CreatedAt != "" {
		fmt.Fprintf(&b, "**Started:** %s\n\n", r.CreatedAt)
	}
	if r.CompletedAt != "" {
		fmt.Fprintf(&b, "**Completed:** %s\n\n", r.CompletedAt)
	}
	if r.TotalCost != "" {
		fmt.Fprintf(&b, "**Cost Estimate:** %s\n\n", r.TotalCost)
	}
	if len(r.Steps) > 0 && r.GatePassRate > 0 {
		fmt.Fprintf(&b, "**Gate Pass Rate:** %.0f%%\n\n", r.GatePassRate*100)
	}

	if len(r.Steps) > 0 {
		b.WriteString("## Steps\n\n")
		b.WriteString("| Step | Role | Status | Duration | Gates |\n")
		b.WriteString("|------|------|--------|----------|-------|\n")
		for _, s := range r.Steps {
			duration := s.Duration
			if duration == "" {
				duration = "—"
			}
			gates := s.Gates
			if gates == "" {
				gates = "—"
			}
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s |\n",
				s.Step, s.Role, s.Status, duration, gates)
		}
		b.WriteString("\n")

		// Handoff summaries.
		hasHandoffs := false
		for _, s := range r.Steps {
			if s.Handoff != "" {
				hasHandoffs = true
				break
			}
		}
		if hasHandoffs {
			b.WriteString("## Handoff Summaries\n\n")
			for _, s := range r.Steps {
				if s.Handoff == "" {
					continue
				}
				fmt.Fprintf(&b, "**Step %d — %s:** %s\n\n", s.Step, s.Role, s.Handoff)
			}
		}
	}

	return b.String()
}

// RenderJSON serialises the report as indented JSON.
func RenderJSON(r *Report) ([]byte, error) {
	// Use a flat map so the JSON key is snake_case "workflow_id" as specified.
	type jsonReport struct {
		WorkflowID   string       `json:"workflow_id"`
		Goal         string       `json:"goal"`
		Status       string       `json:"status"`
		CreatedAt    string       `json:"created_at"`
		CompletedAt  string       `json:"completed_at,omitempty"`
		Steps        []StepReport `json:"steps"`
		GatePassRate float64      `json:"gate_pass_rate"`
		TotalCost    string       `json:"total_cost,omitempty"`
	}
	jr := jsonReport{
		WorkflowID:   r.WorkflowID,
		Goal:         r.Goal,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		CompletedAt:  r.CompletedAt,
		Steps:        r.Steps,
		GatePassRate: r.GatePassRate,
		TotalCost:    r.TotalCost,
	}
	return json.MarshalIndent(jr, "", "  ")
}

// RenderHTML returns a self-contained HTML document with inline CSS.
func RenderHTML(r *Report) string {
	var b strings.Builder

	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Workflow Report</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
  h1 { border-bottom: 2px solid #0366d6; padding-bottom: 8px; }
  h2 { margin-top: 32px; color: #0366d6; }
  table { border-collapse: collapse; width: 100%; margin-top: 12px; }
  th { background: #f6f8fa; text-align: left; padding: 8px 12px; border: 1px solid #ddd; }
  td { padding: 8px 12px; border: 1px solid #ddd; vertical-align: top; }
  tr:nth-child(even) { background: #f9f9f9; }
  .meta { display: grid; grid-template-columns: auto 1fr; gap: 4px 16px; margin-bottom: 24px; }
  .label { font-weight: 600; white-space: nowrap; }
  .badge-passed { background: #28a745; color: #fff; border-radius: 3px; padding: 1px 6px; font-size: 0.85em; }
  .badge-failed { background: #cb2431; color: #fff; border-radius: 3px; padding: 1px 6px; font-size: 0.85em; }
  .badge-completed { background: #0366d6; color: #fff; border-radius: 3px; padding: 1px 6px; font-size: 0.85em; }
  .badge-active { background: #f9a825; color: #333; border-radius: 3px; padding: 1px 6px; font-size: 0.85em; }
</style>
</head>
<body>
`)

	fmt.Fprintf(&b, "<h1>Workflow Report: %s</h1>\n", htmlEsc(r.WorkflowID))
	b.WriteString(`<div class="meta">`)
	writeHTMLMeta(&b, "Goal", r.Goal)
	writeHTMLMeta(&b, "Status", r.Status)
	if r.CreatedAt != "" {
		writeHTMLMeta(&b, "Started", r.CreatedAt)
	}
	if r.CompletedAt != "" {
		writeHTMLMeta(&b, "Completed", r.CompletedAt)
	}
	if r.TotalCost != "" {
		writeHTMLMeta(&b, "Cost Estimate", r.TotalCost)
	}
	if len(r.Steps) > 0 && r.GatePassRate > 0 {
		writeHTMLMeta(&b, "Gate Pass Rate", fmt.Sprintf("%.0f%%", r.GatePassRate*100))
	}
	b.WriteString("</div>\n")

	if len(r.Steps) > 0 {
		b.WriteString("<h2>Steps</h2>\n")
		b.WriteString("<table>\n<tr><th>Step</th><th>Role</th><th>Status</th><th>Duration</th><th>Gates</th><th>Handoff</th></tr>\n")
		for _, s := range r.Steps {
			dur := s.Duration
			if dur == "" {
				dur = "—"
			}
			gateCell := "—"
			if s.Gates == "passed" {
				gateCell = `<span class="badge-passed">passed</span>`
			} else if s.Gates == "failed" {
				gateCell = `<span class="badge-failed">failed</span>`
			}
			statusCell := fmt.Sprintf(`<span class="badge-%s">%s</span>`, htmlEsc(s.Status), htmlEsc(s.Status))
			handoffCell := htmlEsc(s.Handoff)
			if handoffCell == "" {
				handoffCell = "—"
			}
			fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				s.Step, htmlEsc(s.Role), statusCell, htmlEsc(dur), gateCell, handoffCell)
		}
		b.WriteString("</table>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func writeHTMLMeta(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, `<span class="label">%s:</span><span>%s</span>`, htmlEsc(label), htmlEsc(value))
}

// htmlEsc escapes s for safe inclusion in HTML text/attribute contexts.
func htmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}
