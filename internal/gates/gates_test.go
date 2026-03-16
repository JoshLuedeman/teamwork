package gates_test

import (
"os"
"path/filepath"
"testing"

"github.com/joshluedeman/teamwork/internal/gates"
)

// writeScript creates an executable shell script in dir and returns its path.
func writeScript(t *testing.T, dir, name, body string) string {
t.Helper()
path := filepath.Join(dir, name)
if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
t.Fatalf("writeScript: %v", err)
}
return path
}

func TestRunGate_Passes(t *testing.T) {
dir := t.TempDir()
script := writeScript(t, dir, "pass.sh", "exit 0")

passed, output, err := gates.RunGate(dir, script)

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if !passed {
t.Errorf("expected gate to pass, got output: %q", output)
}
}

func TestRunGate_Fails(t *testing.T) {
dir := t.TempDir()
script := writeScript(t, dir, "fail.sh", "echo 'lint error'; exit 1")

passed, output, err := gates.RunGate(dir, script)

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if passed {
t.Error("expected gate to fail, but it passed")
}
if output == "" {
t.Error("expected non-empty output from failing gate")
}
}

func TestRunGate_CapturesOutput(t *testing.T) {
dir := t.TempDir()
script := writeScript(t, dir, "output.sh", "echo 'hello from gate'")

_, output, err := gates.RunGate(dir, script)

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if output != "hello from gate" {
t.Errorf("unexpected output: %q", output)
}
}

func TestRunGate_UsesWorkingDir(t *testing.T) {
dir := t.TempDir()
// Create a sentinel file so the script can assert it runs in the right dir.
if err := os.WriteFile(filepath.Join(dir, "sentinel.txt"), []byte(""), 0o644); err != nil {
t.Fatal(err)
}
script := writeScript(t, dir, "pwd_check.sh", "test -f sentinel.txt")

passed, _, err := gates.RunGate(dir, script)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if !passed {
t.Error("gate should have found sentinel.txt in working dir")
}
}

func TestRunAll_AllPass(t *testing.T) {
dir := t.TempDir()
s1 := writeScript(t, dir, "a.sh", "exit 0")
s2 := writeScript(t, dir, "b.sh", "exit 0")

failures, err := gates.RunAll(dir, []string{s1, s2})

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(failures) != 0 {
t.Errorf("expected no failures, got %d: %+v", len(failures), failures)
}
}

func TestRunAll_MixedPassFail(t *testing.T) {
dir := t.TempDir()
pass := writeScript(t, dir, "pass.sh", "exit 0")
fail1 := writeScript(t, dir, "fail1.sh", "echo 'err1'; exit 1")
fail2 := writeScript(t, dir, "fail2.sh", "echo 'err2'; exit 2")

failures, err := gates.RunAll(dir, []string{pass, fail1, fail2})

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(failures) != 2 {
t.Fatalf("expected 2 failures, got %d: %+v", len(failures), failures)
}
if failures[0].Script != fail1 {
t.Errorf("expected first failure script %q, got %q", fail1, failures[0].Script)
}
if failures[1].Script != fail2 {
t.Errorf("expected second failure script %q, got %q", fail2, failures[1].Script)
}
}

func TestRunAll_Empty(t *testing.T) {
dir := t.TempDir()

failures, err := gates.RunAll(dir, nil)

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(failures) != 0 {
t.Errorf("expected no failures for empty scripts, got %d", len(failures))
}
}

func TestRunAll_CollectsAllFailures(t *testing.T) {
dir := t.TempDir()
// Ensure RunAll doesn't short-circuit after first failure.
s0 := writeScript(t, dir, "f0.sh", "exit 1")
s1 := writeScript(t, dir, "f1.sh", "exit 1")
s2 := writeScript(t, dir, "f2.sh", "exit 1")

failures, err := gates.RunAll(dir, []string{s0, s1, s2})

if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if len(failures) != 3 {
t.Errorf("expected 3 failures, got %d", len(failures))
}
}
