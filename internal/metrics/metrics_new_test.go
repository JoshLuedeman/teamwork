package metrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogDefect(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "metrics"), 0o755); err != nil {
		t.Fatal(err)
	}

	wfID := "feature/1-test"
	if err := LogDefect(dir, wfID, 3, "tester", "Found null pointer bug", "tester"); err != nil {
		t.Fatalf("LogDefect: %v", err)
	}

	events, err := Load(dir, wfID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Action != ActionDefect {
		t.Errorf("action = %q, want %q", events[0].Action, ActionDefect)
	}
	if events[0].DefectSource != "tester" {
		t.Errorf("defect_source = %q, want %q", events[0].DefectSource, "tester")
	}
}

func TestLogWithCost(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".teamwork", "metrics"), 0o755); err != nil {
		t.Fatal(err)
	}

	wfID := "feature/2-cost"
	if err := LogWithCost(dir, wfID, 1, "coder", "Implemented auth", 300, "~$0.05 / 2.3k tokens"); err != nil {
		t.Fatalf("LogWithCost: %v", err)
	}

	events, err := Load(dir, wfID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].CostEstimate != "~$0.05 / 2.3k tokens" {
		t.Errorf("cost_estimate = %q, want %q", events[0].CostEstimate, "~$0.05 / 2.3k tokens")
	}
	if events[0].DurationSec != 300 {
		t.Errorf("duration_sec = %d, want 300", events[0].DurationSec)
	}
}

func TestSummarizeWithDefectsAndCosts(t *testing.T) {
	events := []Event{
		{Workflow: "test/1", Step: 1, Role: "coder", Action: ActionComplete, DurationSec: 100, CostEstimate: "$0.10"},
		{Workflow: "test/1", Step: 2, Role: "tester", Action: ActionDefect, DefectSource: "tester", Detail: "Bug found"},
		{Workflow: "test/1", Step: 3, Role: "reviewer", Action: ActionDefect, DefectSource: "production", Detail: "Escaped bug"},
		{Workflow: "test/1", Step: 4, Role: "coder", Action: ActionComplete, DurationSec: 50, CostEstimate: "$0.05"},
	}

	s := Summarize(events)

	if s.DefectCount != 2 {
		t.Errorf("DefectCount = %d, want 2", s.DefectCount)
	}
	if s.DefectsBySource["tester"] != 1 {
		t.Errorf("DefectsBySource[tester] = %d, want 1", s.DefectsBySource["tester"])
	}
	if s.DefectsBySource["production"] != 1 {
		t.Errorf("DefectsBySource[production] = %d, want 1", s.DefectsBySource["production"])
	}
	if s.TotalCost != "$0.10 + $0.05" {
		t.Errorf("TotalCost = %q, want %q", s.TotalCost, "$0.10 + $0.05")
	}
	if s.TotalDuration != 150 {
		t.Errorf("TotalDuration = %d, want 150", s.TotalDuration)
	}

	rate := s.DefectEscapeRate()
	if rate != 0.5 {
		t.Errorf("DefectEscapeRate = %f, want 0.5", rate)
	}
}

func TestDefectEscapeRateZero(t *testing.T) {
	s := &Summary{DefectCount: 0, DefectsBySource: map[string]int{}}
	if rate := s.DefectEscapeRate(); rate != 0 {
		t.Errorf("DefectEscapeRate = %f, want 0", rate)
	}
}
