package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/memory"
	"github.com/spf13/pflag"
)

func executeFeedbackCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Reset feedback subcommand flags.
	for _, c := range feedbackCmd.Commands() {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			f.Value.Set(f.DefValue) //nolint:errcheck
			f.Changed = false
		})
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"feedback", "--dir", dir}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func writeFeedbackEntries(t *testing.T, dir string, entries []memory.FeedbackEntry) {
	t.Helper()
	ff := &memory.FeedbackFile{Entries: entries}
	if err := memory.SaveFeedback(dir, ff); err != nil {
		t.Fatalf("saving feedback: %v", err)
	}
}

func TestFeedbackList_ShowsEntries(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	writeFeedbackEntries(t, dir, []memory.FeedbackEntry{
		{ID: "feedback-001", Date: "2024-01-01", Source: "pr#42", Domain: []string{"feature"}, Feedback: "Add more tests", Status: "open"},
		{ID: "feedback-002", Date: "2024-01-02", Source: "pr#43", Domain: []string{"bugfix"}, Feedback: "Fix null check", Status: "resolved"},
	})

	out, err := executeFeedbackCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "feedback-001") {
		t.Errorf("expected 'feedback-001' in output:\n%s", out)
	}
	if !strings.Contains(out, "feedback-002") {
		t.Errorf("expected 'feedback-002' in output:\n%s", out)
	}
	if !strings.Contains(out, "Add more tests") {
		t.Errorf("expected feedback content in output:\n%s", out)
	}
}

func TestFeedbackList_StatusFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	writeFeedbackEntries(t, dir, []memory.FeedbackEntry{
		{ID: "feedback-001", Date: "2024-01-01", Source: "pr#1", Domain: []string{"feature"}, Feedback: "Open feedback", Status: "open"},
		{ID: "feedback-002", Date: "2024-01-02", Source: "pr#2", Domain: []string{"feature"}, Feedback: "Resolved feedback", Status: "resolved"},
	})

	out, err := executeFeedbackCmd(t, dir, "list", "--status", "open")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "feedback-001") {
		t.Errorf("expected open entry in output:\n%s", out)
	}
	if strings.Contains(out, "feedback-002") {
		t.Errorf("expected resolved entry to be filtered out:\n%s", out)
	}
}

func TestFeedbackResolve_UpdatesStatus(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)
	writeFeedbackEntries(t, dir, []memory.FeedbackEntry{
		{ID: "feedback-001", Date: "2024-01-01", Source: "pr#1", Domain: []string{"feature"}, Feedback: "Needs fixing", Status: "open"},
	})

	out, err := executeFeedbackCmd(t, dir, "resolve", "feedback-001")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "Resolved") {
		t.Errorf("expected 'Resolved' in output:\n%s", out)
	}

	// Verify the status was updated.
	ff, loadErr := memory.LoadFeedback(dir)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	for _, e := range ff.Entries {
		if e.ID == "feedback-001" && e.Status != "resolved" {
			t.Errorf("expected status=resolved, got %s", e.Status)
		}
	}
}

func TestFeedbackList_EmptyNoEntries(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	out, err := executeFeedbackCmd(t, dir, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "No feedback") {
		t.Errorf("expected 'No feedback' message:\n%s", out)
	}
}
