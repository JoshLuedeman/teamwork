package workflow

import (
"fmt"
"os"
"path/filepath"
"strings"
"testing"
"time"

"github.com/joshluedeman/teamwork/internal/config"
"github.com/joshluedeman/teamwork/internal/handoff"
"github.com/joshluedeman/teamwork/internal/state"
)

// writeGateScript creates an executable shell script in dir with the given body
// and returns its absolute path.
func writeGateScript(t *testing.T, dir, name, body string) string {
t.Helper()
path := filepath.Join(dir, name)
if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
t.Fatalf("writeGateScript: %v", err)
}
return path
}

// setupEngineWithGates builds a temp Engine whose ExtraGates are keyed by
// [workflowType][role] and points at real shell scripts.
func setupEngineWithGates(t *testing.T, extraGates map[string]map[string][]string) (*Engine, string) {
t.Helper()
dir := t.TempDir()
for _, sub := range []string{"state/feature", "handoffs", "metrics"} {
if err := os.MkdirAll(filepath.Join(dir, ".teamwork", sub), 0o755); err != nil {
t.Fatalf("mkdir: %v", err)
}
}
cfg := config.Default()
// Disable standard quality gates so tests focus on extra gates only.
cfg.QualityGates.TestsPass = false
cfg.QualityGates.LintPass = false
cfg.Workflows.ExtraGates = extraGates
return &Engine{Dir: dir, Config: cfg}, dir
}

// makeGateArtifact returns a valid handoff artifact for step 1 / planner.
func makeGateArtifact(wfID string) *handoff.Artifact {
return &handoff.Artifact{
WorkflowID: wfID,
Step:       1,
Role:       "planner",
NextRole:   "architect",
Date:       time.Now().Format(time.RFC3339),
Summary:    "gate test handoff",
Context:    "ctx",
}
}

// createGateWorkflow seeds the .teamwork/state tree with an active workflow at
// step 1, role "planner".
func createGateWorkflow(t *testing.T, dir, wfID, wfType string) {
t.Helper()
subDir := filepath.Join(dir, ".teamwork", "state", filepath.Dir(wfID))
if err := os.MkdirAll(subDir, 0o755); err != nil {
t.Fatalf("mkdir state: %v", err)
}
ws := &state.WorkflowState{
ID:          wfID,
Type:        wfType,
Status:      state.StatusActive,
CurrentStep: 1,
CurrentRole: "planner",
Goal:        "test",
Branch:      wfID,
Steps: []state.StepRecord{
{Step: 1, Role: "planner", Started: time.Now().Format(time.RFC3339)},
},
}
if err := ws.Save(dir); err != nil {
t.Fatalf("save state: %v", err)
}
}

func TestExtraGates_AllPass(t *testing.T) {
dir := t.TempDir()
pass1 := writeGateScript(t, dir, "pass1.sh", "exit 0")
pass2 := writeGateScript(t, dir, "pass2.sh", "exit 0")
eg := map[string]map[string][]string{
"feature": {"planner": {pass1, pass2}},
}
e, eDir := setupEngineWithGates(t, eg)
createGateWorkflow(t, eDir, "feature/42", "feature")

err := e.Handoff("feature/42", makeGateArtifact("feature/42"))
if err != nil {
t.Fatalf("Handoff() unexpected error: %v", err)
}

// Workflow should have advanced to step 2.
ws, _ := state.Load(eDir, "feature/42")
if ws.CurrentStep != 2 {
t.Errorf("expected CurrentStep=2, got %d", ws.CurrentStep)
}
}

func TestExtraGates_OneFails_HandoffRejected(t *testing.T) {
dir := t.TempDir()
pass := writeGateScript(t, dir, "pass.sh", "exit 0")
fail := writeGateScript(t, dir, "fail.sh", fmt.Sprintf("echo 'gate output from %s'; exit 1", "fail.sh"))
eg := map[string]map[string][]string{
"feature": {"planner": {pass, fail}},
}
e, eDir := setupEngineWithGates(t, eg)
createGateWorkflow(t, eDir, "feature/43", "feature")

err := e.Handoff("feature/43", makeGateArtifact("feature/43"))
if err == nil {
t.Fatal("Handoff() should have returned an error when an extra gate fails")
}
if !strings.Contains(err.Error(), "extra gate") {
t.Errorf("error %q should mention 'extra gate'", err.Error())
}
if !strings.Contains(err.Error(), "fail.sh") {
t.Errorf("error %q should identify the failed script", err.Error())
}

// Workflow must remain on step 1 (handoff was rejected).
ws, _ := state.Load(eDir, "feature/43")
if ws.CurrentStep != 1 {
t.Errorf("expected CurrentStep=1 after rejection, got %d", ws.CurrentStep)
}
}

func TestExtraGates_NilExtraGates_SkipsSilently(t *testing.T) {
e, eDir := setupEngineWithGates(t, nil)
createGateWorkflow(t, eDir, "feature/44", "feature")

err := e.Handoff("feature/44", makeGateArtifact("feature/44"))
if err != nil {
t.Fatalf("Handoff() with nil ExtraGates: %v", err)
}
}

func TestExtraGates_WrongWorkflowType_SkipsSilently(t *testing.T) {
dir := t.TempDir()
alwaysFail := writeGateScript(t, dir, "fail.sh", "exit 1")
// Gate is configured for "bugfix" but workflow is "feature".
eg := map[string]map[string][]string{
"bugfix": {"planner": {alwaysFail}},
}
e, eDir := setupEngineWithGates(t, eg)
createGateWorkflow(t, eDir, "feature/45", "feature")

err := e.Handoff("feature/45", makeGateArtifact("feature/45"))
if err != nil {
t.Fatalf("Handoff() should succeed when no gates match workflow type: %v", err)
}
}

func TestExtraGates_WrongRole_SkipsSilently(t *testing.T) {
dir := t.TempDir()
alwaysFail := writeGateScript(t, dir, "fail.sh", "exit 1")
// Gate is configured for "coder" but current role is "planner".
eg := map[string]map[string][]string{
"feature": {"coder": {alwaysFail}},
}
e, eDir := setupEngineWithGates(t, eg)
createGateWorkflow(t, eDir, "feature/46", "feature")

err := e.Handoff("feature/46", makeGateArtifact("feature/46"))
if err != nil {
t.Fatalf("Handoff() should succeed when no gates match current role: %v", err)
}
}
