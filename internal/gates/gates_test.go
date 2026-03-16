package gates

import (
"fmt"
"os/exec"
"testing"
)

type mockRunner struct {
calls   []string
outputs []string
errs    []error
idx     int
}

func (m *mockRunner) Run(command, dir string) ([]byte, error) {
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

func TestRunGateAllPass(t *testing.T) {
m := &mockRunner{outputs: []string{"ok1", "ok2"}, errs: []error{nil, nil}}
results, passed, err := RunGate("after_step_1", []string{"cmd1", "cmd2"}, "/tmp", m)
if err != nil { t.Fatalf("unexpected: %v", err) }
if !passed { t.Fatal("expected pass") }
if len(results) != 2 { t.Fatalf("got %d", len(results)) }
if len(m.calls) != 2 { t.Fatalf("got %d calls", len(m.calls)) }
}

func TestRunGateFirstFails(t *testing.T) {
m := &mockRunner{outputs: []string{"fail"}, errs: []error{&exec.ExitError{}}}
results, passed, err := RunGate("after_step_1", []string{"bad", "good"}, "/tmp", m)
if err != nil { t.Fatalf("unexpected: %v", err) }
if passed { t.Fatal("expected failure") }
if len(results) != 1 { t.Fatalf("got %d", len(results)) }
if len(m.calls) != 1 { t.Fatalf("got %d calls", len(m.calls)) }
}

func TestRunGateSecondFails(t *testing.T) {
m := &mockRunner{outputs: []string{"ok", "fail"}, errs: []error{nil, &exec.ExitError{}}}
_, passed, err := RunGate("after_step_1", []string{"good", "bad"}, "/tmp", m)
if err != nil { t.Fatalf("unexpected: %v", err) }
if passed { t.Fatal("expected failure") }
}

func TestRunGateEmptyConditions(t *testing.T) {
m := &mockRunner{}
results, passed, err := RunGate("after_step_1", []string{}, "/tmp", m)
if err != nil { t.Fatalf("unexpected: %v", err) }
if !passed { t.Fatal("should pass") }
if len(results) != 0 { t.Fatalf("got %d", len(results)) }
}

func TestGateKey(t *testing.T) {
if got := GateKey(1); got != "after_step_1" { t.Errorf("got %q", got) }
if got := GateKey(3); got != "after_step_3" { t.Errorf("got %q", got) }
}

func TestLookup(t *testing.T) {
g := map[string]map[string][]string{"feature": {"after_step_2": {"lint", "test"}}}
if got := Lookup(g, "feature", "after_step_2"); len(got) != 2 { t.Fatalf("got %v", got) }
if Lookup(g, "bugfix", "after_step_2") != nil { t.Fatal("wrong type") }
if Lookup(g, "feature", "after_step_1") != nil { t.Fatal("wrong loc") }
if Lookup(nil, "feature", "after_step_2") != nil { t.Fatal("nil") }
}

func TestShellRunnerIntegration(t *testing.T) {
r := ShellRunner{}
out, err := r.Run("echo hello", t.TempDir())
if err != nil { t.Fatalf("unexpected: %v", err) }
if string(out) != "hello\n" { t.Errorf("got %q", string(out)) }
_, err = r.Run("exit 1", t.TempDir())
if err == nil { t.Fatal("expected error") }
if _, ok := err.(*exec.ExitError); !ok { t.Fatalf("got %T", err) }
out, err = r.Run(fmt.Sprintf("echo fail && exit 1"), t.TempDir())
if err == nil { t.Fatal("expected error") }
if string(out) != "fail\n" { t.Errorf("got %q", string(out)) }
}
