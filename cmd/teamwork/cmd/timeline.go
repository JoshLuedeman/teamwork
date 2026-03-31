package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joshluedeman/teamwork/internal/metrics"
	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/cobra"
)

var timelineCmd = &cobra.Command{
	Use:   "timeline <workflow-id>",
	Short: "Show a visual timeline of workflow steps",
	Long:  "Display an ASCII timeline of workflow steps with status, duration, and handoff information.\nUse --mermaid to emit a Mermaid Gantt diagram instead.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTimeline,
}

func init() {
	timelineCmd.Flags().Bool("mermaid", false, "Emit a Mermaid Gantt diagram instead of an ASCII table")
	rootCmd.AddCommand(timelineCmd)
}

func runTimeline(cmd *cobra.Command, args []string) error {
	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		return err
	}
	workflowID := args[0]
	mermaid, _ := cmd.Flags().GetBool("mermaid")

	ws, err := state.Load(dir, workflowID)
	if err != nil {
		return fmt.Errorf("timeline: load state for %q: %w", workflowID, err)
	}

	// Load metrics events for duration data; ignore error if no metrics file exists.
	events, _ := metrics.Load(dir, workflowID)
	durations := buildDurationMap(events)

	w := cmd.OutOrStdout()
	if mermaid {
		return renderMermaid(w, ws)
	}
	return renderTimelineTable(w, ws, durations)
}

// buildDurationMap returns a map of step number → duration in seconds derived
// from ActionComplete metrics events.
func buildDurationMap(events []metrics.Event) map[int]int {
	m := make(map[int]int)
	for _, e := range events {
		if e.Action == metrics.ActionComplete && e.DurationSec > 0 {
			m[e.Step] = e.DurationSec
		}
	}
	return m
}

// isTerminal reports whether os.Stdout is attached to a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// colorize wraps s in ANSI escape codes when stdout is a terminal.
func colorize(s, code string) string {
	if !isTerminal() {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// stepStatusLabel returns a status string with ANSI color for the given step.
func stepStatusLabel(sr state.StepRecord, currentStep int) string {
	if sr.Completed != "" {
		return colorize("✅ completed", "32")
	}
	if sr.Step == currentStep {
		return colorize("🔄 active", "33")
	}
	return colorize("⏳ pending", "90")
}

// timelineDuration formats a duration given in seconds as a human-readable string.
// Returns "—" for zero or negative durations.
func timelineDuration(secs int) string {
	if secs <= 0 {
		return "—"
	}
	d := time.Duration(secs) * time.Second
	if d >= time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// renderTimelineTable prints an ASCII table timeline to w.
func renderTimelineTable(w interface{ Write([]byte) (int, error) }, ws *state.WorkflowState, durations map[int]int) error {
	fmt.Fprintf(w, "Workflow: %s  Type: %s  Status: %s\n\n", ws.ID, ws.Type, ws.Status)
	fmt.Fprintf(w, "%-6s  %-16s  %-20s  %-10s  %s\n", "Step", "Role", "Status", "Duration", "Handoff File")
	fmt.Fprintln(w, "------  ----------------  --------------------  ----------  --------------------")

	for _, sr := range ws.Steps {
		dur := durations[sr.Step]
		// Show duration only for completed steps; show "—" for pending.
		if sr.Completed == "" {
			dur = 0
		}
		durStr := timelineDuration(dur)

		handoff := sr.Handoff
		if handoff == "" {
			handoff = "—"
		}

		statusStr := stepStatusLabel(sr, ws.CurrentStep)
		fmt.Fprintf(w, "%-6d  %-16s  %-20s  %-10s  %s\n",
			sr.Step, sr.Role, statusStr, durStr, handoff)
	}
	return nil
}

// renderMermaid emits a Mermaid Gantt diagram to w using step start/end times.
func renderMermaid(w interface{ Write([]byte) (int, error) }, ws *state.WorkflowState) error {
	fmt.Fprintln(w, "gantt")
	fmt.Fprintf(w, "    title Workflow: %s\n", ws.ID)
	fmt.Fprintln(w, "    dateFormat YYYY-MM-DDTHH:mm:ss")

	for _, sr := range ws.Steps {
		if sr.Started == "" {
			continue
		}
		// Strip timezone suffix for Mermaid compatibility.
		start := stripTZ(sr.Started)
		end := sr.Completed
		if end == "" {
			end = time.Now().UTC().Format("2006-01-02T15:04:05")
		} else {
			end = stripTZ(end)
		}

		taskStatus := "active"
		if sr.Completed != "" {
			taskStatus = "done"
		}

		fmt.Fprintf(w, "    %s (step %d) :%s, %s, %s\n", sr.Role, sr.Step, taskStatus, start, end)
	}
	return nil
}

// stripTZ removes the trailing "Z" or "+00:00" timezone suffix from an
// RFC3339 timestamp so it is compatible with Mermaid's dateFormat.
func stripTZ(ts string) string {
	ts = strings.TrimSuffix(ts, "Z")
	if idx := strings.LastIndex(ts, "+"); idx > 10 {
		ts = ts[:idx]
	}
	return ts
}
