// Package metrics logs and reports agent activity metrics as JSONL files.
//
// Metrics are stored in .teamwork/metrics/ with one file per workflow instance.
// Each line is a JSON object representing a single event (start, complete, fail, etc.).
// Slashes in workflow IDs are replaced with double underscores in filenames.
// These files are gitignored — they are local runtime data for reporting.
package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Action constants define the valid values for Event.Action.
const (
	ActionStart       = "start"
	ActionComplete    = "complete"
	ActionFail        = "fail"
	ActionEscalate    = "escalate"
	ActionQualityGate = "quality_gate"
	ActionBlock       = "block"
	ActionUnblock     = "unblock"
	ActionDefect      = "defect"
	ActionCancel      = "cancel"
)

// Event represents a single metrics entry logged during workflow execution.
type Event struct {
	Timestamp    string `json:"ts"`
	Workflow     string `json:"workflow"`
	Step         int    `json:"step"`
	Role         string `json:"role"`
	Action       string `json:"action"`
	Detail       string `json:"detail"`
	DurationSec  int    `json:"duration_sec,omitempty"`
	Result       string `json:"result,omitempty"`
	CostEstimate string `json:"cost_estimate,omitempty"`
	Error        string `json:"error,omitempty"`
	DefectSource string `json:"defect_source,omitempty"`
}

// Summary aggregates metrics events for a single workflow into a high-level report.
type Summary struct {
	WorkflowID      string
	TotalDuration   int            // seconds
	StepCount       int
	FailureCount    int
	EscalationCount int
	DefectCount     int
	DefectsBySource map[string]int // defect_source → count
	RoleDurations   map[string]int // role → total seconds
	TotalCost       string         // aggregated cost estimate
}

// Log appends a JSONL line to the metrics file for the given workflow.
// It creates the file and parent directories if they do not exist.
// If event.Timestamp is empty, it is set to the current time in UTC.
func Log(dir, workflowID string, event Event) error {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if event.Workflow == "" {
		event.Workflow = workflowID
	}

	p := metricsPath(dir, workflowID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create metrics directory: %w", err)
	}

	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open metrics file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	data = append(data, '\n')

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	return nil
}

// LogStart logs a start event for the given workflow step and role.
func LogStart(dir, workflowID string, step int, role, detail string) error {
	return Log(dir, workflowID, Event{
		Step:   step,
		Role:   role,
		Action: ActionStart,
		Detail: detail,
	})
}

// LogComplete logs a completion event with the duration in seconds.
func LogComplete(dir, workflowID string, step int, role, detail string, durationSec int) error {
	return Log(dir, workflowID, Event{
		Step:        step,
		Role:        role,
		Action:      ActionComplete,
		Detail:      detail,
		DurationSec: durationSec,
	})
}

// LogFail logs a failure event with an error message.
func LogFail(dir, workflowID string, step int, role, detail, errMsg string) error {
	return Log(dir, workflowID, Event{
		Step:   step,
		Role:   role,
		Action: ActionFail,
		Detail: detail,
		Error:  errMsg,
	})
}

// LogGate logs a quality gate result (passed or failed).
func LogGate(dir, workflowID string, step int, role, detail, result string) error {
	return Log(dir, workflowID, Event{
		Step:   step,
		Role:   role,
		Action: ActionQualityGate,
		Detail: detail,
		Result: result,
	})
}

// LogDefect logs a defect event with the source where the defect was found.
// Valid defect sources: tester, reviewer, security-auditor, production, user.
func LogDefect(dir, workflowID string, step int, role, detail, defectSource string) error {
	return Log(dir, workflowID, Event{
		Step:         step,
		Role:         role,
		Action:       ActionDefect,
		Detail:       detail,
		DefectSource: defectSource,
	})
}

// LogWithCost logs a completion event that includes a cost estimate string.
func LogWithCost(dir, workflowID string, step int, role, detail string, durationSec int, cost string) error {
	return Log(dir, workflowID, Event{
		Step:         step,
		Role:         role,
		Action:       ActionComplete,
		Detail:       detail,
		DurationSec:  durationSec,
		CostEstimate: cost,
	})
}

// Load reads all events from the JSONL metrics file for the given workflow.
// It returns an empty slice (not an error) if the file does not exist.
func Load(dir, workflowID string) ([]Event, error) {
	p := metricsPath(dir, workflowID)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open metrics file: %w", err)
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read metrics file: %w", err)
	}
	return events, nil
}

// Summarize aggregates a slice of events into a Summary.
func Summarize(events []Event) *Summary {
	if len(events) == 0 {
		return &Summary{RoleDurations: make(map[string]int), DefectsBySource: make(map[string]int)}
	}

	s := &Summary{
		WorkflowID:      events[0].Workflow,
		RoleDurations:   make(map[string]int),
		DefectsBySource: make(map[string]int),
	}

	steps := make(map[int]bool)
	var costs []string
	for _, ev := range events {
		steps[ev.Step] = true

		switch ev.Action {
		case ActionComplete:
			s.TotalDuration += ev.DurationSec
			s.RoleDurations[ev.Role] += ev.DurationSec
			if ev.CostEstimate != "" {
				costs = append(costs, ev.CostEstimate)
			}
		case ActionFail:
			s.FailureCount++
			s.TotalDuration += ev.DurationSec
			s.RoleDurations[ev.Role] += ev.DurationSec
		case ActionEscalate:
			s.EscalationCount++
		case ActionDefect:
			s.DefectCount++
			if ev.DefectSource != "" {
				s.DefectsBySource[ev.DefectSource]++
			}
		}
	}
	s.StepCount = len(steps)

	if len(costs) > 0 {
		s.TotalCost = strings.Join(costs, " + ")
	}

	return s
}

// DefectEscapeRate returns the ratio of production defects to total defects.
// Returns 0 if there are no defects.
func (s *Summary) DefectEscapeRate() float64 {
	if s.DefectCount == 0 {
		return 0
	}
	return float64(s.DefectsBySource["production"]) / float64(s.DefectCount)
}

// SummarizeAll reads all JSONL files in .teamwork/metrics/ and returns
// a summary for each workflow.
func SummarizeAll(dir string) ([]*Summary, error) {
	metricsDir := filepath.Join(dir, ".teamwork", "metrics")
	entries, err := os.ReadDir(metricsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read metrics directory: %w", err)
	}

	var summaries []*Summary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		// Recover workflow ID from filename: replace __ with / and strip .jsonl.
		name := strings.TrimSuffix(entry.Name(), ".jsonl")
		workflowID := strings.ReplaceAll(name, "__", "/")

		events, err := Load(dir, workflowID)
		if err != nil {
			return nil, fmt.Errorf("load events for %s: %w", workflowID, err)
		}
		summaries = append(summaries, Summarize(events))
	}
	return summaries, nil
}

// metricsPath returns the file path for a workflow's metrics JSONL file.
// Slashes in workflowID are replaced with double underscores.
func metricsPath(dir, workflowID string) string {
	safe := strings.ReplaceAll(workflowID, "/", "__")
	return filepath.Join(dir, ".teamwork", "metrics", safe+".jsonl")
}
