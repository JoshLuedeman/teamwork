package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JoshLuedeman/teamwork/internal/metrics"
)

func writeTestEvents(t *testing.T, dir, workflowID string, events []metrics.Event) {
	t.Helper()
	for i := range events {
		if events[i].Workflow == "" {
			events[i].Workflow = workflowID
		}
	}
	safe := strings.ReplaceAll(workflowID, "/", "__")
	metricsDir := filepath.Join(dir, ".teamwork", "metrics")
	if err := os.MkdirAll(metricsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(metricsDir, safe+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(data)
		f.Write([]byte("\n"))
	}
}

func executeLogsCmd(t *testing.T, args ...string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	logsCmd.Flags().Set("role", "")
	logsCmd.Flags().Set("action", "")
	logsCmd.Flags().Set("tail", "0")
	logsCmd.Flags().Set("since", "")
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, buf.String())
	}
	return buf.String()
}

func TestLogs_NoMetricsDirectory(t *testing.T) {
	dir := t.TempDir()
	output := executeLogsCmd(t, "logs", "--dir", dir)
	if !strings.Contains(output, "No metrics directory found") {
		t.Errorf("expected helpful message about missing directory, got: %q", output)
	}
}

func TestLogs_EmptyMetricsDirectory(t *testing.T) {
	dir := t.TempDir()
	metricsDir := filepath.Join(dir, ".teamwork", "metrics")
	if err := os.MkdirAll(metricsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	output := executeLogsCmd(t, "logs", "--dir", dir)
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected empty output, got: %q", output)
	}
}

func TestLogs_ParseAndDisplayEvents(t *testing.T) {
	dir := t.TempDir()
	events := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:15Z", Action: "start", Role: "planner", Step: 2, Detail: "Decompose goal"},
		{Timestamp: "2026-03-08T10:31:42Z", Action: "complete", Role: "planner", Step: 2, Detail: "Step 2 done (87s)", DurationSec: 87},
	}
	writeTestEvents(t, dir, "feature/add-oauth", events)
	output := executeLogsCmd(t, "logs", "--dir", dir)
	if !strings.Contains(output, "2026-03-08 10:30:15") {
		t.Errorf("expected formatted timestamp, got: %q", output)
	}
	if !strings.Contains(output, "feature/add-oauth") {
		t.Errorf("expected workflow ID, got: %q", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "start") {
		t.Errorf("first line should be start event, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "complete") {
		t.Errorf("second line should be complete event, got: %q", lines[1])
	}
}

func TestLogs_RoleFilter(t *testing.T) {
	dir := t.TempDir()
	events := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:15Z", Action: "start", Role: "planner", Step: 1, Detail: "Planning"},
		{Timestamp: "2026-03-08T10:31:42Z", Action: "start", Role: "coder", Step: 2, Detail: "Coding"},
		{Timestamp: "2026-03-08T10:32:00Z", Action: "start", Role: "tester", Step: 3, Detail: "Testing"},
	}
	writeTestEvents(t, dir, "test-wf", events)
	output := executeLogsCmd(t, "logs", "--dir", dir, "--role", "coder")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with role filter, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "coder") {
		t.Errorf("expected coder entry, got: %q", lines[0])
	}
}

func TestLogs_ActionFilter(t *testing.T) {
	dir := t.TempDir()
	events := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:15Z", Action: "start", Role: "planner", Step: 1, Detail: "Starting"},
		{Timestamp: "2026-03-08T10:31:42Z", Action: "complete", Role: "planner", Step: 1, Detail: "Done"},
		{Timestamp: "2026-03-08T10:32:00Z", Action: "fail", Role: "coder", Step: 2, Detail: "Failed"},
	}
	writeTestEvents(t, dir, "test-wf", events)
	output := executeLogsCmd(t, "logs", "--dir", dir, "--action", "complete")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with action filter, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "complete") {
		t.Errorf("expected complete entry, got: %q", lines[0])
	}
}

func TestLogs_TailLimit(t *testing.T) {
	dir := t.TempDir()
	events := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:00Z", Action: "start", Role: "planner", Step: 1, Detail: "First"},
		{Timestamp: "2026-03-08T10:31:00Z", Action: "start", Role: "coder", Step: 2, Detail: "Second"},
		{Timestamp: "2026-03-08T10:32:00Z", Action: "start", Role: "tester", Step: 3, Detail: "Third"},
		{Timestamp: "2026-03-08T10:33:00Z", Action: "start", Role: "reviewer", Step: 4, Detail: "Fourth"},
		{Timestamp: "2026-03-08T10:34:00Z", Action: "complete", Role: "reviewer", Step: 4, Detail: "Fifth"},
	}
	writeTestEvents(t, dir, "test-wf", events)
	output := executeLogsCmd(t, "logs", "--dir", dir, "--tail", "2")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines with --tail 2, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "Fourth") {
		t.Errorf("first tail line should be Fourth, got: %q", lines[0])
	}
	if !strings.Contains(lines[1], "Fifth") {
		t.Errorf("second tail line should be Fifth, got: %q", lines[1])
	}
}

func TestLogs_SinceDurationFilter(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	oldTime := now.Add(-48 * time.Hour).Format(time.RFC3339)
	recentTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
	events := []metrics.Event{
		{Timestamp: oldTime, Action: "start", Role: "planner", Step: 1, Detail: "Old event"},
		{Timestamp: recentTime, Action: "start", Role: "coder", Step: 2, Detail: "Recent event"},
	}
	writeTestEvents(t, dir, "test-wf", events)
	output := executeLogsCmd(t, "logs", "--dir", dir, "--since", "24h")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with --since 24h, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "Recent event") {
		t.Errorf("expected recent event, got: %q", lines[0])
	}
}

func TestLogs_WorkflowIDFilter(t *testing.T) {
	dir := t.TempDir()
	eventsA := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:00Z", Action: "start", Role: "planner", Step: 1, Detail: "Workflow A"},
	}
	eventsB := []metrics.Event{
		{Timestamp: "2026-03-08T10:31:00Z", Action: "start", Role: "coder", Step: 1, Detail: "Workflow B"},
	}
	writeTestEvents(t, dir, "workflow-a", eventsA)
	writeTestEvents(t, dir, "workflow-b", eventsB)
	output := executeLogsCmd(t, "logs", "workflow-a", "--dir", dir)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line for specific workflow, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "Workflow A") {
		t.Errorf("expected Workflow A entry, got: %q", lines[0])
	}
}

func TestLogs_OutputFormat(t *testing.T) {
	dir := t.TempDir()
	events := []metrics.Event{
		{Timestamp: "2026-03-08T10:30:15Z", Action: "start", Role: "planner", Step: 2, Detail: "Step 2: Decompose goal into tasks"},
	}
	writeTestEvents(t, dir, "feature/add-oauth", events)
	output := executeLogsCmd(t, "logs", "--dir", dir)
	expected := "2026-03-08 10:30:15  feature/add-oauth         start         planner     Step 2: Decompose goal into tasks"
	line := strings.TrimSpace(output)
	if line != expected {
		t.Errorf("output format mismatch:\nwant: %q\ngot:  %q", expected, line)
	}
}

func TestLogs_CombinedFilters(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	recentTime := now.Add(-1 * time.Hour).Format(time.RFC3339)
	oldTime := now.Add(-48 * time.Hour).Format(time.RFC3339)
	events := []metrics.Event{
		{Timestamp: oldTime, Action: "start", Role: "coder", Step: 1, Detail: "Old coder start"},
		{Timestamp: recentTime, Action: "start", Role: "coder", Step: 2, Detail: "Recent coder start"},
		{Timestamp: recentTime, Action: "complete", Role: "coder", Step: 2, Detail: "Recent coder complete"},
		{Timestamp: recentTime, Action: "start", Role: "tester", Step: 3, Detail: "Recent tester start"},
	}
	writeTestEvents(t, dir, "test-wf", events)
	output := executeLogsCmd(t, "logs", "--dir", dir, "--role", "coder", "--action", "start", "--since", "24h")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with combined filters, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "Recent coder start") {
		t.Errorf("expected recent coder start, got: %q", lines[0])
	}
}

func TestParseSince_Durations(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"1h"},
		{"24h"},
		{"7d"},
		{"30d"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSince(tt.input)
			if err != nil {
				t.Fatalf("parseSince(%q) returned error: %v", tt.input, err)
			}
			if result.IsZero() {
				t.Fatalf("parseSince(%q) returned zero time", tt.input)
			}
			if result.After(time.Now().UTC()) {
				t.Errorf("parseSince(%q) returned future time", tt.input)
			}
		})
	}
}

func TestParseSince_ISODates(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"2026-03-01"},
		{"2026-03-01T10:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSince(tt.input)
			if err != nil {
				t.Fatalf("parseSince(%q) returned error: %v", tt.input, err)
			}
			if result.IsZero() {
				t.Fatalf("parseSince(%q) returned zero time", tt.input)
			}
		})
	}
}

func TestParseSince_Invalid(t *testing.T) {
	_, err := parseSince("not-a-date")
	if err == nil {
		t.Error("expected error for invalid since value")
	}
}
