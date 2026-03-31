package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "View aggregate workflow analytics",
}

var analyticsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show aggregate workflow summary statistics",
	RunE:  runAnalyticsSummary,
}

func init() {
	analyticsSummaryCmd.Flags().String("since", "", "Only include workflows created after this duration ago (e.g. 24h, 7d)")
	analyticsSummaryCmd.Flags().String("type", "", "Filter by workflow type")
	analyticsSummaryCmd.Flags().String("format", "", "Output format: json")
	analyticsCmd.AddCommand(analyticsSummaryCmd)
	rootCmd.AddCommand(analyticsCmd)
}

// analyticsJSON is the JSON output structure for analytics summary.
type analyticsJSON struct {
	Total           int                    `json:"total"`
	Completed       int                    `json:"completed"`
	Failed          int                    `json:"failed"`
	Active          int                    `json:"active"`
	Cancelled       int                    `json:"cancelled"`
	QualityGateRate float64                `json:"quality_gate_pass_rate"`
	EscalationRate  float64                `json:"escalation_rate"`
	ByType          map[string]typeMetrics `json:"by_type"`
}

type typeMetrics struct {
	Count       int     `json:"count"`
	Completed   int     `json:"completed"`
	AvgDuration float64 `json:"avg_duration_sec"`
}

func runAnalyticsSummary(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}

	since, err := cmd.Flags().GetString("since")
	if err != nil {
		return err
	}

	wfType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	// Parse --since duration.
	var sinceTime time.Time
	if since != "" {
		d, parseErr := parseDuration(since)
		if parseErr != nil {
			return fmt.Errorf("invalid --since value: %w", parseErr)
		}
		sinceTime = time.Now().Add(-d)
	}

	workflows, err := state.LoadAll(dir)
	if err != nil {
		// If the state directory doesn't exist yet, treat it as no workflows.
		if os.IsNotExist(err) {
			workflows = nil
		} else {
			return fmt.Errorf("loading workflows: %w", err)
		}
	}

	// Apply filters.
	var filtered []*state.WorkflowState
	for _, ws := range workflows {
		if wfType != "" && ws.Type != wfType {
			continue
		}
		if !sinceTime.IsZero() {
			created, parseErr := time.Parse(time.RFC3339, ws.CreatedAt)
			if parseErr != nil || created.Before(sinceTime) {
				continue
			}
		}
		filtered = append(filtered, ws)
	}

	// Count statuses.
	counts := map[string]int{
		state.StatusActive:    0,
		state.StatusCompleted: 0,
		state.StatusFailed:    0,
		state.StatusCancelled: 0,
	}
	for _, ws := range filtered {
		counts[ws.Status]++
	}

	// Per-type breakdown.
	typeMap := make(map[string]*typeMetrics)
	for _, ws := range filtered {
		tm, ok := typeMap[ws.Type]
		if !ok {
			tm = &typeMetrics{}
			typeMap[ws.Type] = tm
		}
		tm.Count++
		if ws.Status == state.StatusCompleted {
			tm.Completed++
			// Calculate duration between CreatedAt and last completed step.
			dur := workflowDuration(ws)
			tm.AvgDuration += dur
		}
	}
	// Finalize averages.
	for _, tm := range typeMap {
		if tm.Completed > 0 {
			tm.AvgDuration /= float64(tm.Completed)
		}
	}

	// Quality gate pass rate.
	gateTotal, gatePassed := 0, 0
	for _, ws := range filtered {
		for _, step := range ws.Steps {
			if step.QualityGate != "" {
				gateTotal++
				if step.QualityGate == "passed" {
					gatePassed++
				}
			}
		}
	}
	var gateRate float64
	if gateTotal > 0 {
		gateRate = float64(gatePassed) / float64(gateTotal)
	}

	// Escalation rate: completed workflows with at least one escalated blocker.
	escalated := 0
	completed := counts[state.StatusCompleted]
	for _, ws := range filtered {
		for _, b := range ws.Blockers {
			if b.EscalatedTo != "" {
				escalated++
				break
			}
		}
	}
	var escalationRate float64
	if completed > 0 {
		escalationRate = float64(escalated) / float64(completed)
	}

	total := len(filtered)

	if format == "json" {
		// Build JSON-serializable map.
		byTypeJSON := make(map[string]typeMetrics, len(typeMap))
		for k, v := range typeMap {
			byTypeJSON[k] = *v
		}
		out := analyticsJSON{
			Total:           total,
			Completed:       counts[state.StatusCompleted],
			Failed:          counts[state.StatusFailed],
			Active:          counts[state.StatusActive],
			Cancelled:       counts[state.StatusCancelled],
			QualityGateRate: gateRate,
			EscalationRate:  escalationRate,
			ByType:          byTypeJSON,
		}
		data, marshalErr := json.MarshalIndent(out, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("marshaling JSON: %w", marshalErr)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Human-readable output.
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Workflows: %d total, %d completed, %d failed, %d active, %d cancelled\n",
		total, counts[state.StatusCompleted], counts[state.StatusFailed],
		counts[state.StatusActive], counts[state.StatusCancelled])
	fmt.Fprintf(w, "Quality gate pass rate: %.1f%%\n", gateRate*100)
	fmt.Fprintf(w, "Escalation rate: %.1f%%\n", escalationRate*100)

	if len(typeMap) > 0 {
		fmt.Fprintf(w, "\nPer-type breakdown:\n")
		fmt.Fprintf(w, "  %-20s  %5s  %9s  %12s\n", "Type", "Total", "Completed", "Avg Duration")
		fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 55))
		for t, tm := range typeMap {
			fmt.Fprintf(w, "  %-20s  %5d  %9d  %12s\n",
				t, tm.Count, tm.Completed, formatDuration(int(tm.AvgDuration)))
		}
	}

	return nil
}

// workflowDuration returns the duration in seconds between CreatedAt and the
// last completed step timestamp.
func workflowDuration(ws *state.WorkflowState) float64 {
	created, err := time.Parse(time.RFC3339, ws.CreatedAt)
	if err != nil {
		return 0
	}

	var lastCompleted time.Time
	for _, step := range ws.Steps {
		if step.Completed == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, step.Completed)
		if err != nil {
			continue
		}
		if t.After(lastCompleted) {
			lastCompleted = t
		}
	}

	if lastCompleted.IsZero() {
		return 0
	}
	return lastCompleted.Sub(created).Seconds()
}

// parseDuration extends time.ParseDuration to support "Nd" for N days.
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		var days float64
		if _, err := fmt.Sscanf(n, "%f", &days); err != nil {
			return 0, fmt.Errorf("invalid days value %q", s)
		}
		return time.Duration(days * 24 * float64(time.Hour)), nil
	}
	return time.ParseDuration(s)
}
