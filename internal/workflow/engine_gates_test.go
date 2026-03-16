package workflow

import (
"os"
"os/exec"
"path/filepath"
"strings"
"testing"
"time"

"github.com/joshluedeman/teamwork/internal/config"
"github.com/joshluedeman/teamwork/internal/gates"
"github.com/joshluedeman/teamwork/internal/handoff"
"github.com/joshluedeman/teamwork/internal/state"
)

func realExitError() *exec.ExitError {
err := exec.Command("/bin/sh", "-c", "exit 1").Run()
ee, _ := err.(*exec.ExitError)
return ee
}

type mockGateRunner struct {
calls   []string
outputs []string
errs    []error
idx     int
}

func (m *mockGateRunner) Run(command, dir string) ([]byte, error) {
m.calls = append(m.calls, command)
i := m.idx
m.idx++
var out string
if i < len(m.outputs) {
out = m.outputs[i]
}
if i < len(m.errs) && m.errs[i] != nil {
return []byte(out), m.errs[i]
}
return []byte(out), nil
}

func setupEngineWithGates(t *testing.T, extraGates map[string]map[string][]string, runner gates.Runner) (*Engine, string) {
t.Helper()
dir := t.TempDir()
twDir := filepath.Join(dir, ".teamwork")
if err := os.MkdirAll(twDir, 0o755); err != nil { t.Fatal(err) }
cfgYAML := "project_name: test\nworkflows:\n  extra_gates: {}\n"
if err := os.WriteFile(filepath.Join(twDir, "config.yaml"), []byte(cfgYAML), 0o644); err != nil { t.Fatal(err) }
if err := os.MkdirAll(filepath.Join(twDir, "metrics"), 0o755); err != nil { t.Fatal(err) }
if err := os.MkdirAll(filepath.Join(twDir, "handoffs"), 0o755); err != nil { t.Fatal(err) }
cfg, err := config.Load(dir)
if err != nil { t.Fatal(err) }
cfg.Workflows.ExtraGates = extraGates
return &Engine{Dir: dir, Config: cfg, GateRunner: runner}, dir
}

func makeTestArtifact(wfID string, step int, role, nextRole string) *handoff.Artifact {
return &handoff.Artifact{
WorkflowID: wfID, Step: step, Role: role, NextRole: nextRole,
Date: time.Now().Format(time.RFC3339), Summary: "test", Context: "ctx",
}
}

func createTestWorkflow(t *testing.T, dir, wfID, wfType string, step int, role string) {
t.Helper()
ws := &state.WorkflowState{
ID: wfID, Type: wfType, Status: state.StatusActive,
CurrentStep: step, CurrentRole: role, Goal: "test", Branch: wfID,
Steps: []state.StepRecord{{Step: step, Role: role, Started: time.Now().Format(time.RFC3339)}},
}
if err := ws.Save(dir); err != nil { t.Fatal(err) }
}

func TestExtraGatesPass(t *testing.T) {
r := &mockGateRunner{outputs: []string{"ok1", "ok2"}, errs: []error{nil, nil}}
eg := map[string]map[string][]string{"feature": {"after_step_1": {"lint", "test"}}}
e, dir := setupEngineWithGates(t, eg, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err != nil { t.Fatalf("expected success: %v", err) }
if len(r.calls) != 2 { t.Fatalf("expected 2 calls, got %d", len(r.calls)) }
ws, _ := state.Load(dir, "feature-42-x")
if ws.CurrentStep != 2 { t.Fatalf("expected step 2, got %d", ws.CurrentStep) }
}

func TestExtraGatesFail(t *testing.T) {
r := &mockGateRunner{outputs: []string{"ok", "FAIL"}, errs: []error{nil, realExitError()}}
eg := map[string]map[string][]string{"feature": {"after_step_1": {"lint", "test"}}}
e, dir := setupEngineWithGates(t, eg, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err == nil { t.Fatal("expected error") }
if !strings.Contains(err.Error(), "extra gate") { t.Fatalf("wrong error: %v", err) }
ws, _ := state.Load(dir, "feature-42-x")
if ws.CurrentStep != 1 { t.Fatalf("expected step 1, got %d", ws.CurrentStep) }
}

func TestExtraGatesNoGatesConfigured(t *testing.T) {
r := &mockGateRunner{}
e, dir := setupEngineWithGates(t, nil, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err != nil { t.Fatalf("expected success: %v", err) }
if len(r.calls) != 0 { t.Fatalf("expected 0 calls, got %d", len(r.calls)) }
}

func TestExtraGatesWrongStep(t *testing.T) {
r := &mockGateRunner{}
eg := map[string]map[string][]string{"feature": {"after_step_2": {"no"}}}
e, dir := setupEngineWithGates(t, eg, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err != nil { t.Fatalf("expected success: %v", err) }
if len(r.calls) != 0 { t.Fatalf("expected 0 calls, got %d", len(r.calls)) }
}

func TestExtraGatesWrongWorkflowType(t *testing.T) {
r := &mockGateRunner{}
eg := map[string]map[string][]string{"bugfix": {"after_step_1": {"no"}}}
e, dir := setupEngineWithGates(t, eg, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err != nil { t.Fatalf("expected success: %v", err) }
if len(r.calls) != 0 { t.Fatalf("expected 0 calls, got %d", len(r.calls)) }
}

func TestExtraGatesFirstCommandFails(t *testing.T) {
r := &mockGateRunner{outputs: []string{"FAIL"}, errs: []error{realExitError()}}
eg := map[string]map[string][]string{"feature": {"after_step_1": {"bad", "good"}}}
e, dir := setupEngineWithGates(t, eg, r)
createTestWorkflow(t, dir, "feature-42-x", "feature", 1, "planner")
err := e.Handoff("feature-42-x", makeTestArtifact("feature-42-x", 1, "planner", "coder"))
if err == nil { t.Fatal("expected error") }
if len(r.calls) != 1 { t.Fatalf("expected 1 call, got %d", len(r.calls)) }
if r.calls[0] != "bad" { t.Fatalf("expected bad, got %q", r.calls[0]) }
}
