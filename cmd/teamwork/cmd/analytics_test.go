package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joshluedeman/teamwork/internal/state"
	"github.com/spf13/pflag"
)

func executeAnalyticsCmd(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Reset flag state to avoid pollution between tests.
	analyticsSummaryCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue) //nolint:errcheck
		f.Changed = false
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(append([]string{"analytics", "summary", "--dir", dir}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// writeStateYAML writes a workflow state YAML file for analytics tests.
func writeStateYAML(t *testing.T, dir string, ws *state.WorkflowState) {
	t.Helper()
	stateDir := filepath.Join(dir, ".teamwork", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(dir); err != nil {
		t.Fatalf("saving state: %v", err)
	}
}

// buildAnalyticsFixture creates 3 completed + 1 failed + 1 active workflow states.
func buildAnalyticsFixture(t *testing.T, dir string) {
	t.Helper()
	writeTestConfig(t, dir, minimalConfig)

	completed1 := state.New("feature/1-auth", "feature", "Implement auth")
	completed1.Status = state.StatusCompleted
	completed1.Steps = []state.StepRecord{
		{Step: 1, Role: "human", QualityGate: "passed", Completed: "2024-01-02T00:00:00Z"},
		{Step: 2, Role: "coder", QualityGate: "passed", Completed: "2024-01-03T00:00:00Z"},
	}
	completed1.CreatedAt = "2024-01-01T00:00:00Z"
	writeStateYAML(t, dir, completed1)

	completed2 := state.New("feature/2-api", "feature", "Build API")
	completed2.Status = state.StatusCompleted
	completed2.Steps = []state.StepRecord{
		{Step: 1, Role: "human", QualityGate: "passed", Completed: "2024-01-05T00:00:00Z"},
	}
	completed2.CreatedAt = "2024-01-04T00:00:00Z"
	writeStateYAML(t, dir, completed2)

	completed3 := state.New("bugfix/1-crash", "bugfix", "Fix crash")
	completed3.Status = state.StatusCompleted
	completed3.Steps = []state.StepRecord{
		{Step: 1, Role: "human", QualityGate: "passed", Completed: "2024-01-10T00:00:00Z"},
	}
	completed3.CreatedAt = "2024-01-09T00:00:00Z"
	writeStateYAML(t, dir, completed3)

	failed1 := state.New("feature/3-failed", "feature", "Failed feature")
	failed1.Status = state.StatusFailed
	failed1.Steps = []state.StepRecord{
		{Step: 1, Role: "coder", QualityGate: "failed"},
	}
	writeStateYAML(t, dir, failed1)

	active1 := state.New("feature/4-active", "feature", "Active feature")
	active1.Status = state.StatusActive
	writeStateYAML(t, dir, active1)
}

func TestAnalyticsSummary_CorrectCounts(t *testing.T) {
	dir := t.TempDir()
	buildAnalyticsFixture(t, dir)

	out, err := executeAnalyticsCmd(t, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "5 total") {
		t.Errorf("expected '5 total' in output:\n%s", out)
	}
	if !strings.Contains(out, "3 completed") {
		t.Errorf("expected '3 completed' in output:\n%s", out)
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("expected '1 failed' in output:\n%s", out)
	}
	if !strings.Contains(out, "1 active") {
		t.Errorf("expected '1 active' in output:\n%s", out)
	}
}

func TestAnalyticsSummary_TypeFilter(t *testing.T) {
	dir := t.TempDir()
	buildAnalyticsFixture(t, dir)

	out, err := executeAnalyticsCmd(t, dir, "--type", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should show 4 feature workflows (3 completed/failed/active), not 5
	if strings.Contains(out, "5 total") {
		t.Errorf("expected type filter to exclude bugfix workflows:\n%s", out)
	}
	if !strings.Contains(out, "4 total") {
		t.Errorf("expected '4 total' (only feature) in output:\n%s", out)
	}
}

func TestAnalyticsSummary_JSONFormat(t *testing.T) {
	dir := t.TempDir()
	buildAnalyticsFixture(t, dir)

	out, err := executeAnalyticsCmd(t, dir, "--format", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result analyticsJSON
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if result.Completed != 3 {
		t.Errorf("expected completed=3, got %d", result.Completed)
	}
	if result.Failed != 1 {
		t.Errorf("expected failed=1, got %d", result.Failed)
	}
	if result.Active != 1 {
		t.Errorf("expected active=1, got %d", result.Active)
	}
	if _, ok := result.ByType["feature"]; !ok {
		t.Error("expected 'feature' key in by_type")
	}
	if _, ok := result.ByType["bugfix"]; !ok {
		t.Error("expected 'bugfix' key in by_type")
	}
}

func TestAnalyticsSummary_NoWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, minimalConfig)

	out, err := executeAnalyticsCmd(t, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "0 total") {
		t.Errorf("expected '0 total' in output:\n%s", out)
	}
}

func TestParseDuration_Days(t *testing.T) {
	d, err := parseDuration("7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 7 * 24 * 60 * 60 // seconds
	if int(d.Seconds()) != expected {
		t.Errorf("expected %d seconds, got %d", expected, int(d.Seconds()))
	}
}

func TestParseDuration_Hours(t *testing.T) {
	d, err := parseDuration("24h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Hours() != 24 {
		t.Errorf("expected 24h, got %v", d)
	}
}
